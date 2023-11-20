import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { environment } from '../../environments/environment';

@Injectable({
  providedIn: 'root'
})
export class example {

  constructor(private http: HttpClient) {}

  // @ts-ignore
  public async HasPermission(payload: HasPermissionRequest): Promise<HasPermissionResponse> {
    // tslint:disable-next-line:max-line-length
    return await this.http.post(environment.BackendURL + '/example/haspermission', JSON.stringify(payload)).toPromise() as HasPermissionResponse;
  }

  // @ts-ignore
  public async WhatsTheTime(payload: WhatsTheTimeRequest): Promise<WhatsTheTimeResponse> {
    // tslint:disable-next-line:max-line-length
    return await this.http.post(environment.BackendURL + '/example/whatsthetime', JSON.stringify(payload)).toPromise() as WhatsTheTimeResponse;
  }
}

export interface HasPermissionRequest {
  R: Role[];
  U: User;
  InventoryUpdate: Record<number, boolean>;
}

export interface HasPermissionResponse {
  Bool: boolean;
}

export interface WhatsTheTimeRequest {
  Date: Date;
  Toy: second.Toy;
}

export interface WhatsTheTimeResponse {
  Bool: boolean;
}

export interface Toy {
  Design: string;
  CreatedAt: Date;
}

export interface Toy {
  Design: string;
  CreatedAt: Date;
}

export interface User {
  ID: number;
  Name: string;
  Role: Role;
  T: Toy;
}

export enum Role {
  RoleAdmin = 2,
  RoleUnknown = 0,
  RoleUser = 1,
}
