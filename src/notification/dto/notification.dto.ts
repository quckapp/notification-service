import { ApiProperty, ApiPropertyOptional } from '@nestjs/swagger';
import {
  IsString,
  IsUUID,
  IsEnum,
  IsOptional,
  IsObject,
  IsDateString,
  MaxLength,
  IsUrl,
} from 'class-validator';
import {
  NotificationType,
  NotificationPriority,
} from '../entities/notification.entity';

export class SendNotificationDto {
  @ApiProperty()
  @IsUUID()
  userId: string;

  @ApiPropertyOptional()
  @IsUUID()
  @IsOptional()
  workspaceId?: string;

  @ApiProperty({ enum: NotificationType })
  @IsEnum(NotificationType)
  type: NotificationType;

  @ApiProperty()
  @IsString()
  @MaxLength(255)
  title: string;

  @ApiProperty()
  @IsString()
  body: string;

  @ApiPropertyOptional()
  @IsObject()
  @IsOptional()
  data?: Record<string, any>;

  @ApiPropertyOptional({ enum: NotificationPriority })
  @IsEnum(NotificationPriority)
  @IsOptional()
  priority?: NotificationPriority;

  @ApiPropertyOptional()
  @IsString()
  @IsOptional()
  @MaxLength(100)
  category?: string;

  @ApiPropertyOptional()
  @IsUrl()
  @IsOptional()
  actionUrl?: string;

  @ApiPropertyOptional()
  @IsUrl()
  @IsOptional()
  imageUrl?: string;

  @ApiPropertyOptional()
  @IsUUID()
  @IsOptional()
  templateId?: string;

  @ApiPropertyOptional()
  @IsDateString()
  @IsOptional()
  scheduledAt?: string;

  @ApiPropertyOptional()
  @IsDateString()
  @IsOptional()
  expiresAt?: string;
}

export class SendBulkNotificationDto {
  @ApiProperty({ type: [String] })
  @IsUUID('4', { each: true })
  userIds: string[];

  @ApiPropertyOptional()
  @IsUUID()
  @IsOptional()
  workspaceId?: string;

  @ApiProperty({ enum: NotificationType })
  @IsEnum(NotificationType)
  type: NotificationType;

  @ApiProperty()
  @IsString()
  @MaxLength(255)
  title: string;

  @ApiProperty()
  @IsString()
  body: string;

  @ApiPropertyOptional()
  @IsObject()
  @IsOptional()
  data?: Record<string, any>;

  @ApiPropertyOptional({ enum: NotificationPriority })
  @IsEnum(NotificationPriority)
  @IsOptional()
  priority?: NotificationPriority;
}

export class MarkReadDto {
  @ApiProperty({ type: [String] })
  @IsUUID('4', { each: true })
  notificationIds: string[];
}

export class NotificationResponseDto {
  @ApiProperty()
  id: string;

  @ApiProperty()
  userId: string;

  @ApiPropertyOptional()
  workspaceId?: string;

  @ApiProperty({ enum: NotificationType })
  type: NotificationType;

  @ApiProperty()
  status: string;

  @ApiProperty()
  priority: string;

  @ApiProperty()
  title: string;

  @ApiProperty()
  body: string;

  @ApiPropertyOptional()
  data?: Record<string, any>;

  @ApiPropertyOptional()
  category?: string;

  @ApiPropertyOptional()
  actionUrl?: string;

  @ApiPropertyOptional()
  imageUrl?: string;

  @ApiPropertyOptional()
  sentAt?: Date;

  @ApiPropertyOptional()
  readAt?: Date;

  @ApiProperty()
  createdAt: Date;
}
