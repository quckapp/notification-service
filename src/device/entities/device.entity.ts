import {
  Entity,
  PrimaryGeneratedColumn,
  Column,
  CreateDateColumn,
  UpdateDateColumn,
  Index,
} from 'typeorm';

export enum DevicePlatform {
  IOS = 'ios',
  ANDROID = 'android',
  WEB = 'web',
}

@Entity('device_tokens')
@Index(['userId', 'platform'])
@Index(['token'], { unique: true })
export class DeviceToken {
  @PrimaryGeneratedColumn('uuid')
  id: string;

  @Column('uuid')
  @Index()
  userId: string;

  @Column({
    type: 'enum',
    enum: DevicePlatform,
  })
  platform: DevicePlatform;

  @Column({ length: 500 })
  token: string;

  @Column({ length: 255, nullable: true })
  deviceName: string;

  @Column({ length: 50, nullable: true })
  deviceModel: string;

  @Column({ length: 20, nullable: true })
  osVersion: string;

  @Column({ length: 20, nullable: true })
  appVersion: string;

  @Column({ default: true })
  isActive: boolean;

  @Column({ nullable: true })
  lastUsedAt: Date;

  @CreateDateColumn()
  createdAt: Date;

  @UpdateDateColumn()
  updatedAt: Date;
}
