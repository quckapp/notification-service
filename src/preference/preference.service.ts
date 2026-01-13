import { Injectable, Logger, NotFoundException } from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import { Repository } from 'typeorm';
import { NotificationPreference } from './entities/preference.entity';
import { NotificationType } from '../notification/entities/notification.entity';

export class UpdatePreferenceDto {
  pushEnabled?: boolean;
  emailEnabled?: boolean;
  smsEnabled?: boolean;
  inAppEnabled?: boolean;
  mentionsEnabled?: boolean;
  directMessagesEnabled?: boolean;
  channelMessagesEnabled?: boolean;
  threadRepliesEnabled?: boolean;
  reactionsEnabled?: boolean;
  systemNotificationsEnabled?: boolean;
  muteAll?: boolean;
  muteUntil?: Date;
  quietHoursStart?: string;
  quietHoursEnd?: string;
  quietHoursEnabled?: boolean;
  timezone?: string;
  mutedChannels?: string[];
}

@Injectable()
export class PreferenceService {
  private readonly logger = new Logger(PreferenceService.name);

  constructor(
    @InjectRepository(NotificationPreference)
    private readonly preferenceRepo: Repository<NotificationPreference>,
  ) {}

  async getOrCreatePreferences(
    userId: string,
    workspaceId?: string,
  ): Promise<NotificationPreference> {
    let preference = await this.preferenceRepo.findOne({
      where: { userId, workspaceId: workspaceId || null },
    });

    if (!preference) {
      preference = this.preferenceRepo.create({
        userId,
        workspaceId,
      });
      await this.preferenceRepo.save(preference);
    }

    return preference;
  }

  async updatePreferences(
    userId: string,
    workspaceId: string | null,
    dto: UpdatePreferenceDto,
  ): Promise<NotificationPreference> {
    let preference = await this.preferenceRepo.findOne({
      where: { userId, workspaceId },
    });

    if (!preference) {
      preference = this.preferenceRepo.create({
        userId,
        workspaceId,
        ...dto,
      });
    } else {
      Object.assign(preference, dto);
    }

    await this.preferenceRepo.save(preference);
    this.logger.log(`Updated preferences for user ${userId}`);
    return preference;
  }

  async canSendNotification(
    userId: string,
    workspaceId: string | undefined,
    type: NotificationType,
  ): Promise<boolean> {
    const preference = await this.getOrCreatePreferences(userId, workspaceId);

    // Check global mute
    if (preference.muteAll) {
      if (!preference.muteUntil || preference.muteUntil > new Date()) {
        return false;
      }
    }

    // Check quiet hours
    if (preference.quietHoursEnabled) {
      const now = new Date();
      const currentTime = `${now.getHours().toString().padStart(2, '0')}:${now.getMinutes().toString().padStart(2, '0')}`;
      if (
        currentTime >= preference.quietHoursStart &&
        currentTime <= preference.quietHoursEnd
      ) {
        // Only block non-urgent notifications during quiet hours
        return false;
      }
    }

    // Check type-specific settings
    switch (type) {
      case NotificationType.PUSH:
        return preference.pushEnabled;
      case NotificationType.EMAIL:
        return preference.emailEnabled;
      case NotificationType.SMS:
        return preference.smsEnabled;
      case NotificationType.IN_APP:
        return preference.inAppEnabled;
      default:
        return true;
    }
  }

  async muteChannel(
    userId: string,
    workspaceId: string,
    channelId: string,
  ): Promise<void> {
    const preference = await this.getOrCreatePreferences(userId, workspaceId);
    const mutedChannels = preference.mutedChannels || [];
    if (!mutedChannels.includes(channelId)) {
      mutedChannels.push(channelId);
      preference.mutedChannels = mutedChannels;
      await this.preferenceRepo.save(preference);
    }
  }

  async unmuteChannel(
    userId: string,
    workspaceId: string,
    channelId: string,
  ): Promise<void> {
    const preference = await this.getOrCreatePreferences(userId, workspaceId);
    if (preference.mutedChannels) {
      preference.mutedChannels = preference.mutedChannels.filter(
        (id) => id !== channelId,
      );
      await this.preferenceRepo.save(preference);
    }
  }

  async isChannelMuted(
    userId: string,
    workspaceId: string,
    channelId: string,
  ): Promise<boolean> {
    const preference = await this.preferenceRepo.findOne({
      where: { userId, workspaceId },
    });
    return preference?.mutedChannels?.includes(channelId) || false;
  }
}
