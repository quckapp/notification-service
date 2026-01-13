import {
  Controller,
  Get,
  Post,
  Delete,
  Body,
  Param,
  Headers,
  ParseUUIDPipe,
} from '@nestjs/common';
import { ApiTags, ApiOperation, ApiHeader } from '@nestjs/swagger';
import { DeviceService, RegisterDeviceDto } from './device.service';
import { DeviceToken } from './entities/device.entity';

@ApiTags('devices')
@Controller('devices')
export class DeviceController {
  constructor(private readonly deviceService: DeviceService) {}

  @Post()
  @ApiOperation({ summary: 'Register a device for push notifications' })
  @ApiHeader({ name: 'X-User-Id', required: true })
  async registerDevice(
    @Body() dto: Omit<RegisterDeviceDto, 'userId'>,
    @Headers('X-User-Id') userId: string,
  ): Promise<DeviceToken> {
    return this.deviceService.registerDevice({ ...dto, userId });
  }

  @Get('user/:userId')
  @ApiOperation({ summary: 'Get user devices' })
  async getUserDevices(
    @Param('userId', ParseUUIDPipe) userId: string,
  ): Promise<DeviceToken[]> {
    return this.deviceService.getUserDevices(userId);
  }

  @Delete(':id')
  @ApiOperation({ summary: 'Deactivate a device' })
  @ApiHeader({ name: 'X-User-Id', required: true })
  async deactivateDevice(
    @Param('id', ParseUUIDPipe) id: string,
    @Headers('X-User-Id') userId: string,
  ): Promise<{ success: boolean }> {
    await this.deviceService.deactivateDevice(id, userId);
    return { success: true };
  }

  @Delete('user/:userId/all')
  @ApiOperation({ summary: 'Deactivate all user devices' })
  async deactivateAllDevices(
    @Param('userId', ParseUUIDPipe) userId: string,
  ): Promise<{ success: boolean }> {
    await this.deviceService.deactivateAllDevices(userId);
    return { success: true };
  }
}
