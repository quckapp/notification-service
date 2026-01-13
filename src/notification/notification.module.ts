import { Module, forwardRef } from '@nestjs/common';
import { TypeOrmModule } from '@nestjs/typeorm';
import { BullModule } from '@nestjs/bull';
import { Notification } from './entities/notification.entity';
import { NotificationService } from './notification.service';
import { NotificationController } from './notification.controller';
import { NotificationProcessor } from './notification.processor';
import { PreferenceModule } from '../preference/preference.module';
import { DeviceModule } from '../device/device.module';

@Module({
  imports: [
    TypeOrmModule.forFeature([Notification]),
    BullModule.registerQueue({
      name: 'notifications',
    }),
    forwardRef(() => PreferenceModule),
    forwardRef(() => DeviceModule),
  ],
  controllers: [NotificationController],
  providers: [NotificationService, NotificationProcessor],
  exports: [NotificationService],
})
export class NotificationModule {}
