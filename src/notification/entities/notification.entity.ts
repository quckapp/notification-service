import {
  Entity,
  PrimaryGeneratedColumn,
  Column,
  CreateDateColumn,
  UpdateDateColumn,
  Index,
} from 'typeorm';

export enum NotificationType {
  PUSH = 'push',
  EMAIL = 'email',
  SMS = 'sms',
  IN_APP = 'in_app',
}

export enum NotificationStatus {
  PENDING = 'pending',
  QUEUED = 'queued',
  SENT = 'sent',
  DELIVERED = 'delivered',
  FAILED = 'failed',
  READ = 'read',
}

export enum NotificationPriority {
  LOW = 'low',
  NORMAL = 'normal',
  HIGH = 'high',
  URGENT = 'urgent',
}

@Entity('notifications')
@Index(['userId', 'status'])
@Index(['workspaceId', 'createdAt'])
export class Notification {
  @PrimaryGeneratedColumn('uuid')
  id: string;

  @Column('uuid')
  @Index()
  userId: string;

  @Column('uuid', { nullable: true })
  workspaceId: string;

  @Column({
    type: 'enum',
    enum: NotificationType,
  })
  type: NotificationType;

  @Column({
    type: 'enum',
    enum: NotificationStatus,
    default: NotificationStatus.PENDING,
  })
  status: NotificationStatus;

  @Column({
    type: 'enum',
    enum: NotificationPriority,
    default: NotificationPriority.NORMAL,
  })
  priority: NotificationPriority;

  @Column({ length: 255 })
  title: string;

  @Column('text')
  body: string;

  @Column('jsonb', { nullable: true })
  data: Record<string, any>;

  @Column({ length: 100, nullable: true })
  category: string;

  @Column({ length: 255, nullable: true })
  actionUrl: string;

  @Column({ length: 255, nullable: true })
  imageUrl: string;

  @Column('uuid', { nullable: true })
  templateId: string;

  @Column({ nullable: true })
  scheduledAt: Date;

  @Column({ nullable: true })
  sentAt: Date;

  @Column({ nullable: true })
  deliveredAt: Date;

  @Column({ nullable: true })
  readAt: Date;

  @Column({ nullable: true })
  expiresAt: Date;

  @Column({ length: 255, nullable: true })
  errorMessage: string;

  @Column({ default: 0 })
  retryCount: number;

  @CreateDateColumn()
  createdAt: Date;

  @UpdateDateColumn()
  updatedAt: Date;
}
