from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.host_repute_connection_method import HostReputeConnectionMethod
from ..models.host_repute_connection_status import HostReputeConnectionStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.host_repute_asset import HostReputeAsset
  from ..models.host_repute_scope import HostReputeScope
  from ..models.host_repute_shared import HostReputeShared





T = TypeVar("T", bound="HostReputeConnection")



@_attrs_define
class HostReputeConnection:
    """ 
        Attributes:
            alerts_to_notifications (bool):
            assets (list[HostReputeAsset]):
            check_interval_minutes (int):
            demo (bool):
            monitor_used (int):
            scopes (list[HostReputeScope]):
            shared (HostReputeShared):
            sources (int):
            status (HostReputeConnectionStatus):
            syncing (bool):
            account_email (str | Unset):
            connected_at (datetime.datetime | Unset):
            connected_by (str | Unset):
            dmarc_rua (str | Unset):
            last_check_at (datetime.datetime | Unset):
            last_sync_at (datetime.datetime | Unset):
            method (HostReputeConnectionMethod | Unset):
            monitor_limit (int | Unset):
            plan (str | Unset):
            sync_error (str | Unset):
            sync_error_at (datetime.datetime | Unset):
            team_name (str | Unset):
            token_rejected_at (datetime.datetime | Unset):
     """

    alerts_to_notifications: bool
    assets: list[HostReputeAsset]
    check_interval_minutes: int
    demo: bool
    monitor_used: int
    scopes: list[HostReputeScope]
    shared: HostReputeShared
    sources: int
    status: HostReputeConnectionStatus
    syncing: bool
    account_email: str | Unset = UNSET
    connected_at: datetime.datetime | Unset = UNSET
    connected_by: str | Unset = UNSET
    dmarc_rua: str | Unset = UNSET
    last_check_at: datetime.datetime | Unset = UNSET
    last_sync_at: datetime.datetime | Unset = UNSET
    method: HostReputeConnectionMethod | Unset = UNSET
    monitor_limit: int | Unset = UNSET
    plan: str | Unset = UNSET
    sync_error: str | Unset = UNSET
    sync_error_at: datetime.datetime | Unset = UNSET
    team_name: str | Unset = UNSET
    token_rejected_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.host_repute_asset import HostReputeAsset # noqa: PLC0415
        from ..models.host_repute_scope import HostReputeScope # noqa: PLC0415
        from ..models.host_repute_shared import HostReputeShared # noqa: PLC0415
        alerts_to_notifications = self.alerts_to_notifications

        assets = []
        for assets_item_data in self.assets:
            assets_item = assets_item_data.to_dict()
            assets.append(assets_item)



        check_interval_minutes = self.check_interval_minutes

        demo = self.demo

        monitor_used = self.monitor_used

        scopes = []
        for scopes_item_data in self.scopes:
            scopes_item = scopes_item_data.to_dict()
            scopes.append(scopes_item)



        shared = self.shared.to_dict()

        sources = self.sources

        status = self.status.value

        syncing = self.syncing

        account_email = self.account_email

        connected_at: str | Unset = UNSET
        if not isinstance(self.connected_at, Unset):
            connected_at = self.connected_at.isoformat()

        connected_by = self.connected_by

        dmarc_rua = self.dmarc_rua

        last_check_at: str | Unset = UNSET
        if not isinstance(self.last_check_at, Unset):
            last_check_at = self.last_check_at.isoformat()

        last_sync_at: str | Unset = UNSET
        if not isinstance(self.last_sync_at, Unset):
            last_sync_at = self.last_sync_at.isoformat()

        method: str | Unset = UNSET
        if not isinstance(self.method, Unset):
            method = self.method.value


        monitor_limit = self.monitor_limit

        plan = self.plan

        sync_error = self.sync_error

        sync_error_at: str | Unset = UNSET
        if not isinstance(self.sync_error_at, Unset):
            sync_error_at = self.sync_error_at.isoformat()

        team_name = self.team_name

        token_rejected_at: str | Unset = UNSET
        if not isinstance(self.token_rejected_at, Unset):
            token_rejected_at = self.token_rejected_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "alertsToNotifications": alerts_to_notifications,
            "assets": assets,
            "checkIntervalMinutes": check_interval_minutes,
            "demo": demo,
            "monitorUsed": monitor_used,
            "scopes": scopes,
            "shared": shared,
            "sources": sources,
            "status": status,
            "syncing": syncing,
        })
        if account_email is not UNSET:
            field_dict["accountEmail"] = account_email
        if connected_at is not UNSET:
            field_dict["connectedAt"] = connected_at
        if connected_by is not UNSET:
            field_dict["connectedBy"] = connected_by
        if dmarc_rua is not UNSET:
            field_dict["dmarcRua"] = dmarc_rua
        if last_check_at is not UNSET:
            field_dict["lastCheckAt"] = last_check_at
        if last_sync_at is not UNSET:
            field_dict["lastSyncAt"] = last_sync_at
        if method is not UNSET:
            field_dict["method"] = method
        if monitor_limit is not UNSET:
            field_dict["monitorLimit"] = monitor_limit
        if plan is not UNSET:
            field_dict["plan"] = plan
        if sync_error is not UNSET:
            field_dict["syncError"] = sync_error
        if sync_error_at is not UNSET:
            field_dict["syncErrorAt"] = sync_error_at
        if team_name is not UNSET:
            field_dict["teamName"] = team_name
        if token_rejected_at is not UNSET:
            field_dict["tokenRejectedAt"] = token_rejected_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.host_repute_asset import HostReputeAsset # noqa: PLC0415
        from ..models.host_repute_scope import HostReputeScope # noqa: PLC0415
        from ..models.host_repute_shared import HostReputeShared # noqa: PLC0415
        d = dict(src_dict)
        alerts_to_notifications = d.pop("alertsToNotifications")

        assets = []
        _assets = d.pop("assets")
        for assets_item_data in (_assets):
            assets_item = HostReputeAsset.from_dict(assets_item_data)



            assets.append(assets_item)


        check_interval_minutes = d.pop("checkIntervalMinutes")

        demo = d.pop("demo")

        monitor_used = d.pop("monitorUsed")

        scopes = []
        _scopes = d.pop("scopes")
        for scopes_item_data in (_scopes):
            scopes_item = HostReputeScope.from_dict(scopes_item_data)



            scopes.append(scopes_item)


        shared = HostReputeShared.from_dict(d.pop("shared"))




        sources = d.pop("sources")

        status = HostReputeConnectionStatus(d.pop("status"))




        syncing = d.pop("syncing")

        account_email = d.pop("accountEmail", UNSET)

        _connected_at = d.pop("connectedAt", UNSET)
        connected_at: datetime.datetime | Unset
        if isinstance(_connected_at,  Unset):
            connected_at = UNSET
        else:
            connected_at = datetime.datetime.fromisoformat(_connected_at)




        connected_by = d.pop("connectedBy", UNSET)

        dmarc_rua = d.pop("dmarcRua", UNSET)

        _last_check_at = d.pop("lastCheckAt", UNSET)
        last_check_at: datetime.datetime | Unset
        if isinstance(_last_check_at,  Unset):
            last_check_at = UNSET
        else:
            last_check_at = datetime.datetime.fromisoformat(_last_check_at)




        _last_sync_at = d.pop("lastSyncAt", UNSET)
        last_sync_at: datetime.datetime | Unset
        if isinstance(_last_sync_at,  Unset):
            last_sync_at = UNSET
        else:
            last_sync_at = datetime.datetime.fromisoformat(_last_sync_at)




        _method = d.pop("method", UNSET)
        method: HostReputeConnectionMethod | Unset
        if isinstance(_method,  Unset):
            method = UNSET
        else:
            method = HostReputeConnectionMethod(_method)




        monitor_limit = d.pop("monitorLimit", UNSET)

        plan = d.pop("plan", UNSET)

        sync_error = d.pop("syncError", UNSET)

        _sync_error_at = d.pop("syncErrorAt", UNSET)
        sync_error_at: datetime.datetime | Unset
        if isinstance(_sync_error_at,  Unset):
            sync_error_at = UNSET
        else:
            sync_error_at = datetime.datetime.fromisoformat(_sync_error_at)




        team_name = d.pop("teamName", UNSET)

        _token_rejected_at = d.pop("tokenRejectedAt", UNSET)
        token_rejected_at: datetime.datetime | Unset
        if isinstance(_token_rejected_at,  Unset):
            token_rejected_at = UNSET
        else:
            token_rejected_at = datetime.datetime.fromisoformat(_token_rejected_at)




        host_repute_connection = cls(
            alerts_to_notifications=alerts_to_notifications,
            assets=assets,
            check_interval_minutes=check_interval_minutes,
            demo=demo,
            monitor_used=monitor_used,
            scopes=scopes,
            shared=shared,
            sources=sources,
            status=status,
            syncing=syncing,
            account_email=account_email,
            connected_at=connected_at,
            connected_by=connected_by,
            dmarc_rua=dmarc_rua,
            last_check_at=last_check_at,
            last_sync_at=last_sync_at,
            method=method,
            monitor_limit=monitor_limit,
            plan=plan,
            sync_error=sync_error,
            sync_error_at=sync_error_at,
            team_name=team_name,
            token_rejected_at=token_rejected_at,
        )

        return host_repute_connection

