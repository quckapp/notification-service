import { Injectable, Logger } from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import { Repository, In, LessThanOrEqual } from 'typeorm';
import { InjectQueue } from '@nestjs/bull';
import { Queue } from 'bull';
import { Cron, CronExpression } from '@nestjs/schedule';
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
import { FirebaseProvider } from '../providers/firebase.provider';
import { EmailProvider } from '../providers/email.provider';
import { SmsProvider } from '../providers/sms.provider';

@Injectable()
export class NotificationService {
  private readonly logger = new Logger(NotificationService.name);

  // Rate limiting: track notifications per user
  private readonly userRateLimits = new Map<string, { count: number; resetAt: number }>();
  private readonly RATE_LIMIT_WINDOW = 60 * 1000; // 1 minute
  private readonly RATE_LIMIT_MAX = 100; // max 100 notifications per user per minute

  constructor(
    @InjectRepository(Notification)
    private readonly notificationRepo: Repository<Notification>,
    @InjectQueue('notifications')
    private readonly notificationQueue: Queue,
    private readonly preferenceService: PreferenceService,
    private readonly deviceService: DeviceService,
    private readonly firebaseProvider: FirebaseProvider,
    private readonly emailProvider: EmailProvider,
    private readonly smsProvider: SmsProvider,
  ) {}

  async send(dto: SendNotificationDto): Promise<NotificationResponseDto | null> {
    // Check rate limit
    if (!this.checkRateLimit(dto.userId)) {
      this.logger.warn(`Rate limit exceeded for user: ${dto.userId}`);
      return null;
    }

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
      userId: dto.userId,
      workspaceId: dto.workspaceId,
      type: dto.type,
      title: dto.title,
      body: dto.body,
      data: dto.data,
      priority: dto.priority,
      category: dto.category,
      actionUrl: dto.actionUrl,
      imageUrl: dto.imageUrl,
      status: dto.scheduledAt
        ? NotificationStatus.PENDING
        : NotificationStatus.QUEUED,
      scheduledAt: dto.scheduledAt ? new Date(dto.scheduledAt) : undefined,
      expiresAt: dto.expiresAt ? new Date(dto.expiresAt) : undefined,
    });

    await this.notificationRepo.save(notification);

    // Queue for processing (only if not scheduled)
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
        // Check rate limit
        if (!this.checkRateLimit(userId)) {
          return null;
        }

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

  // Scheduled job to process pending scheduled notifications
  @Cron(CronExpression.EVERY_MINUTE)
  async processScheduledNotifications(): Promise<void> {
    const now = new Date();

    const pendingNotifications = await this.notificationRepo.find({
      where: {
        status: NotificationStatus.PENDING,
        scheduledAt: LessThanOrEqual(now),
      },
      take: 100, // Process max 100 per minute
    });

    if (pendingNotifications.length === 0) {
      return;
    }

    this.logger.log(`Processing ${pendingNotifications.length} scheduled notifications`);

    for (const notification of pendingNotifications) {
      // Check if notification has expired
      if (notification.expiresAt && notification.expiresAt < now) {
        notification.status = NotificationStatus.FAILED;
        notification.errorMessage = 'Notification expired before delivery';
        await this.notificationRepo.save(notification);
        continue;
      }

      // Queue for processing
      notification.status = NotificationStatus.QUEUED;
      await this.notificationRepo.save(notification);

      await this.notificationQueue.add('send', { notificationId: notification.id }, {
        priority: this.getPriorityValue(notification.priority),
        attempts: 3,
        backoff: { type: 'exponential', delay: 1000 },
      });
    }
  }

  async processNotification(notificationId: string): Promise<void> {
    const notification = await this.notificationRepo.findOne({
      where: { id: notificationId },
    });

    if (!notification) {
      this.logger.warn(`Notification not found: ${notificationId}`);
      return;
    }

    // Check if expired
    if (notification.expiresAt && notification.expiresAt < new Date()) {
      notification.status = NotificationStatus.FAILED;
      notification.errorMessage = 'Notification expired';
      await this.notificationRepo.save(notification);
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

    const tokens = devices.map((d) => d.token);

    // Convert data to string values (Firebase requirement)
    const stringData: Record<string, string> = {};
    if (notification.data) {
      for (const [key, value] of Object.entries(notification.data)) {
        stringData[key] = typeof value === 'string' ? value : JSON.stringify(value);
      }
    }

    // Add notification metadata to data
    stringData.notificationId = notification.id;
    if (notification.actionUrl) {
      stringData.actionUrl = notification.actionUrl;
    }

    const result = await this.firebaseProvider.sendToDevices(tokens, {
      title: notification.title,
      body: notification.body,
      data: stringData,
      imageUrl: notification.imageUrl,
      priority: notification.priority === 'urgent' || notification.priority === 'high' ? 'high' : 'normal',
    });

    if (result.success) {
      notification.status = NotificationStatus.SENT;
      this.logger.log(`Push sent to ${result.successCount} devices for notification ${notification.id}`);
    } else if (result.successCount > 0) {
      notification.status = NotificationStatus.SENT;
      notification.errorMessage = `Partial delivery: ${result.failureCount} failed`;
      this.logger.warn(`Partial push delivery for notification ${notification.id}: ${result.failureCount} failed`);
    } else {
      notification.status = NotificationStatus.FAILED;
      notification.errorMessage = result.errors[0]?.error || 'All devices failed';
    }

    // Deactivate failed tokens (invalid/expired)
    if (result.failedTokens.length > 0) {
      this.logger.warn(`Deactivating ${result.failedTokens.length} failed tokens`);
      // Note: In production, you'd want to deactivate these tokens
      // await this.deviceService.deactivateTokens(result.failedTokens);
    }
  }

  private async sendEmailNotification(notification: Notification): Promise<void> {
    // In a real implementation, you'd get the user's email from user-service
    // For now, we'll use the data field or skip
    const email = notification.data?.email as string;

    if (!email) {
      notification.status = NotificationStatus.FAILED;
      notification.errorMessage = 'No email address provided';
      this.logger.warn(`No email address for notification ${notification.id}`);
      return;
    }

    const result = await this.emailProvider.sendEmail({
      to: email,
      subject: notification.title,
      text: notification.body,
      html: notification.data?.htmlBody as string || `<p>${notification.body}</p>`,
    });

    if (result.success) {
      notification.status = NotificationStatus.SENT;
      this.logger.log(`Email sent for notification ${notification.id}: ${result.messageId}`);
    } else {
      notification.status = NotificationStatus.FAILED;
      notification.errorMessage = result.error || 'Email sending failed';
    }
  }

  private async sendSmsNotification(notification: Notification): Promise<void> {
    // In a real implementation, you'd get the user's phone from user-service
    const phone = notification.data?.phone as string;

    if (!phone) {
      notification.status = NotificationStatus.FAILED;
      notification.errorMessage = 'No phone number provided';
      this.logger.warn(`No phone number for notification ${notification.id}`);
      return;
    }

    const result = await this.smsProvider.sendSms({
      to: phone,
      body: `${notification.title}\n\n${notification.body}`,
    });

    if (result.success) {
      notification.status = NotificationStatus.SENT;
      this.logger.log(`SMS sent for notification ${notification.id}: ${result.messageId}`);
    } else {
      notification.status = NotificationStatus.FAILED;
      notification.errorMessage = result.error || 'SMS sending failed';
    }
  }

  // Rate limiting
  private checkRateLimit(userId: string): boolean {
    const now = Date.now();
    const userLimit = this.userRateLimits.get(userId);

    if (!userLimit || userLimit.resetAt < now) {
      // Reset or initialize
      this.userRateLimits.set(userId, {
        count: 1,
        resetAt: now + this.RATE_LIMIT_WINDOW,
      });
      return true;
    }

    if (userLimit.count >= this.RATE_LIMIT_MAX) {
      return false;
    }

    userLimit.count++;
    return true;
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

  // Analytics methods
  async getStats(workspaceId?: string): Promise<{
    total: number;
    sent: number;
    delivered: number;
    failed: number;
    read: number;
    pending: number;
  }> {
    const baseQuery = this.notificationRepo.createQueryBuilder('n');

    if (workspaceId) {
      baseQuery.where('n.workspaceId = :workspaceId', { workspaceId });
    }

    const [total, sent, delivered, failed, read, pending] = await Promise.all([
      baseQuery.clone().getCount(),
      baseQuery.clone().andWhere('n.status = :status', { status: NotificationStatus.SENT }).getCount(),
      baseQuery.clone().andWhere('n.status = :status', { status: NotificationStatus.DELIVERED }).getCount(),
      baseQuery.clone().andWhere('n.status = :status', { status: NotificationStatus.FAILED }).getCount(),
      baseQuery.clone().andWhere('n.status = :status', { status: NotificationStatus.READ }).getCount(),
      baseQuery.clone().andWhere('n.status = :status', { status: NotificationStatus.PENDING }).getCount(),
    ]);

    return { total, sent, delivered, failed, read, pending };
  }

  async getFailedNotifications(limit = 50): Promise<Notification[]> {
    return this.notificationRepo.find({
      where: { status: NotificationStatus.FAILED },
      order: { createdAt: 'DESC' },
      take: limit,
    });
  }

  async retryNotification(notificationId: string): Promise<boolean> {
    const notification = await this.notificationRepo.findOne({
      where: { id: notificationId },
    });

    if (!notification) {
      return false;
    }

    if (notification.status !== NotificationStatus.FAILED) {
      return false;
    }

    notification.status = NotificationStatus.QUEUED;
    notification.errorMessage = null as any;
    await this.notificationRepo.save(notification);

    await this.notificationQueue.add('send', { notificationId }, {
      priority: this.getPriorityValue(notification.priority),
      attempts: 3,
      backoff: { type: 'exponential', delay: 1000 },
    });

    return true;
  }
}
