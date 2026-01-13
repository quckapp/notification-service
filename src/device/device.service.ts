import { Injectable, Logger, NotFoundException } from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import { Repository } from 'typeorm';
import { DeviceToken, DevicePlatform } from './entities/device.entity';

export class RegisterDeviceDto {
  userId: string;
  platform: DevicePlatform;
  token: string;
  deviceName?: string;
  deviceModel?: string;
  osVersion?: string;
  appVersion?: string;
}

@Injectable()
export class DeviceService {
  private readonly logger = new Logger(DeviceService.name);

  constructor(
    @InjectRepository(DeviceToken)
    private readonly deviceRepo: Repository<DeviceToken>,
  ) {}

  async registerDevice(dto: RegisterDeviceDto): Promise<DeviceToken> {
    // Check if token already exists
    let device = await this.deviceRepo.findOne({
      where: { token: dto.token },
    });

    if (device) {
      // Update existing device
      device.userId = dto.userId;
      device.platform = dto.platform;
      device.deviceName = dto.deviceName;
      device.deviceModel = dto.deviceModel;
      device.osVersion = dto.osVersion;
      device.appVersion = dto.appVersion;
      device.isActive = true;
      device.lastUsedAt = new Date();
    } else {
      device = this.deviceRepo.create({
        ...dto,
        isActive: true,
        lastUsedAt: new Date(),
      });
    }

    await this.deviceRepo.save(device);
    this.logger.log(`Registered device for user ${dto.userId}: ${dto.platform}`);
    return device;
  }

  async getUserDevices(userId: string): Promise<DeviceToken[]> {
    return this.deviceRepo.find({
      where: { userId, isActive: true },
      order: { lastUsedAt: 'DESC' },
    });
  }

  async getDeviceById(id: string): Promise<DeviceToken> {
    const device = await this.deviceRepo.findOne({ where: { id } });
    if (!device) {
      throw new NotFoundException('Device not found');
    }
    return device;
  }

  async deactivateDevice(id: string, userId: string): Promise<void> {
    const device = await this.deviceRepo.findOne({
      where: { id, userId },
    });
    if (!device) {
      throw new NotFoundException('Device not found');
    }
    device.isActive = false;
    await this.deviceRepo.save(device);
    this.logger.log(`Deactivated device ${id} for user ${userId}`);
  }

  async deactivateAllDevices(userId: string): Promise<void> {
    await this.deviceRepo.update(
      { userId },
      { isActive: false },
    );
    this.logger.log(`Deactivated all devices for user ${userId}`);
  }

  async updateLastUsed(deviceId: string): Promise<void> {
    await this.deviceRepo.update(deviceId, { lastUsedAt: new Date() });
  }
}
