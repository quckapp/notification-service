import { Injectable, Logger } from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import { Repository, In } from 'typeorm';
import { InjectQueue } from '@nestjs/bull';
import { Queue } from 'bull';
import {
  Notification,
  NotificationStatus,
  NotificationType,
} from './entities/notification.entity';
import {
  SendNotificationDto,
  SendBulkNotificationDto,
  NotificationResponseDto,
} from './dto/notification.dto';
import { PreferenceService } from '../preference/preference.service';
import { DeviceService } from '../device/device.service';

@Injectable()
export class NotificationService {
  private readonly logger = new Logger(NotificationService.name);

  constructor(
    @InjectRepository(Notification)
    private readonly notificationRepo: Repository<Notification>,
    @InjectQueue('notifications')
    private readonly notificationQueue: Queue,
    private readonly preferenceService: PreferenceService,
    private readonly deviceService: DeviceService,
  ) {}

  async send(dto: SendNotificationDto): Promise<NotificationResponseDto> {
    // Check user preferences
    const canSend = await this.preferenceService.canSendNotification(
      dto.userId,
      dto.workspaceId,
      dto.type,
    );

    if (!canSend) {
      this.logger.debug(
        `Notification blocked by user preferences: ${dto.userId}`,
      );
      return null;
    }

    const notification = this.notificationRepo.create({
      ...dto,
      status: dto.scheduledAt
        ? NotificationStatus.PENDING
        : NotificationStatus.QUEUED,
      scheduledAt: dto.scheduledAt ? new Date(dto.scheduledAt) : null,
      expiresAt: dto.expiresAt ? new Date(dto.expiresAt) : null,
    });

    await this.notificationRepo.save(notification);

    // Queue for processing
    if (!dto.scheduledAt) {
      await this.notificationQueue.add('send', { notificationId: notification.id }, {
        priority: this.getPriorityValue(dto.priority),
        attempts: 3,
        backoff: { type: 'exponential', delay: 1000 },
      });
    }

    return this.toResponse(notification);
  }

  async sendBulk(dto: SendBulkNotificationDto): Promise<{ queued: number }> {
    const notifications = await Promise.all(
      dto.userIds.map(async (userId) => {
        const canSend = await this.preferenceService.canSendNotification(
          userId,
          dto.workspaceId,
          dto.type,
        );
        if (!canSend) return null;

        return this.notificationRepo.create({
          userId,
          workspaceId: dto.workspaceId,
          type: dto.type,
          title: dto.title,
          body: dto.body,
          data: dto.data,
          priority: dto.priority,
          status: NotificationStatus.QUEUED,
        });
      }),
    );

    const validNotifications = notifications.filter((n) => n !== null);
    await this.notificationRepo.save(validNotifications);

    // Queue all for processing
    await Promise.all(
      validNotifications.map((n) =>
        this.notificationQueue.add('send', { notificationId: n.id }),
      ),
    );

    return { queued: validNotifications.length };
  }

  async getUserNotifications(
    userId: string,
    workspaceId?: string,
    page = 0,
    limit = 20,
  ): Promise<{ data: NotificationResponseDto[]; total: number }> {
    const query = this.notificationRepo
      .createQueryBuilder('n')
      .where('n.userId = :userId', { userId })
      .andWhere('n.type = :type', { type: NotificationType.IN_APP });

    if (workspaceId) {
      query.andWhere('n.workspaceId = :workspaceId', { workspaceId });
    }

    const [notifications, total] = await query
      .orderBy('n.createdAt', 'DESC')
      .skip(page * limit)
      .take(limit)
      .getManyAndCount();

    return {
      data: notifications.map((n) => this.toResponse(n)),
      total,
    };
  }

  async getUnreadCount(userId: string, workspaceId?: string): Promise<number> {
    const query = this.notificationRepo
      .createQueryBuilder('n')
      .where('n.userId = :userId', { userId })
      .andWhere('n.type = :type', { type: NotificationType.IN_APP })
      .andWhere('n.status != :status', { status: NotificationStatus.READ });

    if (workspaceId) {
      query.andWhere('n.workspaceId = :workspaceId', { workspaceId });
    }

    return query.getCount();
  }

  async markAsRead(userId: string, notificationIds: string[]): Promise<void> {
    await this.notificationRepo.update(
      { id: In(notificationIds), userId },
      { status: NotificationStatus.READ, readAt: new Date() },
    );
  }

  async markAllAsRead(userId: string, workspaceId?: string): Promise<void> {
    const query = this.notificationRepo
      .createQueryBuilder()
      .update(Notification)
      .set({ status: NotificationStatus.READ, readAt: new Date() })
      .where('userId = :userId', { userId })
      .andWhere('status != :status', { status: NotificationStatus.READ });

    if (workspaceId) {
      query.andWhere('workspaceId = :workspaceId', { workspaceId });
    }

    await query.execute();
  }

  async processNotification(notificationId: string): Promise<void> {
    const notification = await this.notificationRepo.findOne({
      where: { id: notificationId },
    });

    if (!notification) {
      this.logger.warn(`Notification not found: ${notificationId}`);
      return;
    }

    try {
      switch (notification.type) {
        case NotificationType.PUSH:
          await this.sendPushNotification(notification);
          break;
        case NotificationType.EMAIL:
          await this.sendEmailNotification(notification);
          break;
        case NotificationType.SMS:
          await this.sendSmsNotification(notification);
          break;
        case NotificationType.IN_APP:
          // In-app notifications are just stored, no external delivery
          notification.status = NotificationStatus.DELIVERED;
          notification.deliveredAt = new Date();
          break;
      }

      notification.sentAt = new Date();
      await this.notificationRepo.save(notification);
    } catch (error) {
      this.logger.error(
        `Failed to send notification ${notificationId}:`,
        error,
      );
      notification.status = NotificationStatus.FAILED;
      notification.errorMessage = error.message;
      notification.retryCount += 1;
      await this.notificationRepo.save(notification);
      throw error;
    }
  }

  private async sendPushNotification(notification: Notification): Promise<void> {
    const devices = await this.deviceService.getUserDevices(notification.userId);
    if (devices.length === 0) {
      this.logger.debug(`No devices found for user: ${notification.userId}`);
      notification.status = NotificationStatus.FAILED;
      notification.errorMessage = 'No registered devices';
      return;
    }

    // Send to all devices (Firebase implementation would go here)
    this.logger.log(`Sending push to ${devices.length} devices for user ${notification.userId}`);
    notification.status = NotificationStatus.SENT;
  }

  private async sendEmailNotification(notification: Notification): Promise<void> {
    // Email sending implementation (Nodemailer would go here)
    this.logger.log(`Sending email notification to user ${notification.userId}`);
    notification.status = NotificationStatus.SENT;
  }

  private async sendSmsNotification(notification: Notification): Promise<void> {
    // SMS sending implementation (Twilio would go here)
    this.logger.log(`Sending SMS notification to user ${notification.userId}`);
    notification.status = NotificationStatus.SENT;
  }

  private getPriorityValue(priority?: string): number {
    switch (priority) {
      case 'urgent':
        return 1;
      case 'high':
        return 2;
      case 'normal':
        return 3;
      case 'low':
        return 4;
      default:
        return 3;
    }
  }

  private toResponse(notification: Notification): NotificationResponseDto {
    return {
      id: notification.id,
      userId: notification.userId,
      workspaceId: notification.workspaceId,
      type: notification.type,
      status: notification.status,
      priority: notification.priority,
      title: notification.title,
      body: notification.body,
      data: notification.data,
      category: notification.category,
      actionUrl: notification.actionUrl,
      imageUrl: notification.imageUrl,
      sentAt: notification.sentAt,
      readAt: notification.readAt,
      createdAt: notification.createdAt,
    };
  }
}
