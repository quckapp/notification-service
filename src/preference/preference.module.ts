import { Module } from '@nestjs/common';
import { TypeOrmModule } from '@nestjs/typeorm';
import { NotificationPreference } from './entities/preference.entity';
import { PreferenceService } from './preference.service';
import { PreferenceController } from './preference.controller';

@Module({
  imports: [TypeOrmModule.forFeature([NotificationPreference])],
  controllers: [PreferenceController],
  providers: [PreferenceService],
  exports: [PreferenceService],
})
export class PreferenceModule {}
