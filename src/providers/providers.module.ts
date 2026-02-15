import { Global, Module } from '@nestjs/common';
import { ConfigModule } from '@nestjs/config';
import { FirebaseProvider } from './firebase.provider';
import { EmailProvider } from './email.provider';
import { SmsProvider } from './sms.provider';

@Global()
@Module({
  imports: [ConfigModule],
  providers: [FirebaseProvider, EmailProvider, SmsProvider],
  exports: [FirebaseProvider, EmailProvider, SmsProvider],
})
export class ProvidersModule {}
