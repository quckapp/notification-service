import {
  Entity,
  PrimaryGeneratedColumn,
  Column,
  CreateDateColumn,
  UpdateDateColumn,
  Index,
} from 'typeorm';

@Entity('notification_preferences')
@Index(['userId', 'workspaceId'], { unique: true })
export class NotificationPreference {
  @PrimaryGeneratedColumn('uuid')
  id: string;

  @Column('uuid')
  @Index()
  userId: string;

  @Column('uuid', { nullable: true })
  workspaceId: string;

  @Column({ default: true })
  pushEnabled: boolean;

  @Column({ default: true })
  emailEnabled: boolean;

  @Column({ default: false })
  smsEnabled: boolean;

  @Column({ default: true })
  inAppEnabled: boolean;

  @Column({ default: true })
  mentionsEnabled: boolean;

  @Column({ default: true })
  directMessagesEnabled: boolean;

  @Column({ default: true })
  channelMessagesEnabled: boolean;

  @Column({ default: true })
  threadRepliesEnabled: boolean;

  @Column({ default: true })
  reactionsEnabled: boolean;

  @Column({ default: true })
  systemNotificationsEnabled: boolean;

  @Column({ default: false })
  muteAll: boolean;

  @Column({ nullable: true })
  muteUntil: Date;

  @Column({ length: 5, default: '09:00' })
  quietHoursStart: string;

  @Column({ length: 5, default: '21:00' })
  quietHoursEnd: string;

  @Column({ default: false })
  quietHoursEnabled: boolean;

  @Column({ length: 50, default: 'UTC' })
  timezone: string;

  @Column('simple-array', { nullable: true })
  mutedChannels: string[];

  @CreateDateColumn()
  createdAt: Date;

  @UpdateDateColumn()
  updatedAt: Date;
}
