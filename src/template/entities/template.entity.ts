import {
  Entity,
  PrimaryGeneratedColumn,
  Column,
  CreateDateColumn,
  UpdateDateColumn,
  Index,
} from 'typeorm';
import { NotificationType } from '../../notification/entities/notification.entity';

@Entity('notification_templates')
@Index(['name'], { unique: true })
export class NotificationTemplate {
  @PrimaryGeneratedColumn('uuid')
  id: string;

  @Column({ length: 100 })
  name: string;

  @Column({ length: 255, nullable: true })
  description: string;

  @Column({
    type: 'enum',
    enum: NotificationType,
  })
  type: NotificationType;

  @Column({ length: 255 })
  subject: string;

  @Column('text')
  bodyTemplate: string;

  @Column('text', { nullable: true })
  htmlTemplate: string;

  @Column('jsonb', { nullable: true })
  defaultData: Record<string, any>;

  @Column({ length: 50, nullable: true })
  category: string;

  @Column({ default: true })
  isActive: boolean;

  @Column('uuid', { nullable: true })
  workspaceId: string;

  @CreateDateColumn()
  createdAt: Date;

  @UpdateDateColumn()
  updatedAt: Date;
}
