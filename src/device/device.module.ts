import { Module } from '@nestjs/common';
import { TypeOrmModule } from '@nestjs/typeorm';
import { DeviceToken } from './entities/device.entity';
import { DeviceService } from './device.service';
import { DeviceController } from './device.controller';

@Module({
  imports: [TypeOrmModule.forFeature([DeviceToken])],
  controllers: [DeviceController],
  providers: [DeviceService],
  exports: [DeviceService],
})
export class DeviceModule {}
