from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.earn_connection_health import EarnConnectionHealth
from ..models.earn_connection_mode import EarnConnectionMode
from ..models.earn_connection_provider import EarnConnectionProvider
from ..models.earn_connection_status import EarnConnectionStatus
from ..models.earn_connection_step import EarnConnectionStep
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.earn_account import EarnAccount
  from ..models.earn_permission_check import EarnPermissionCheck
  from ..models.earn_webhook import EarnWebhook





T = TypeVar("T", bound="EarnConnection")



@_attrs_define
class EarnConnection:
    """ 
        Attributes:
            account (EarnAccount):
            create_key_url (str):
            dashboard_url (str):
            demo (bool):
            emails_url (str):
            granted (int):
            health (EarnConnectionHealth):
            id (str):
            key_hint (str):
            keys_url (str):
            mode (EarnConnectionMode):
            permissions (list[EarnPermissionCheck]):
            portal_set (bool):
            provider (EarnConnectionProvider):
            provider_name (str):
            status (EarnConnectionStatus):
            step (EarnConnectionStep):
            webhook (EarnWebhook):
            webhooks_url (str):
            checked_at (datetime.datetime | Unset):
            connected_at (datetime.datetime | Unset):
            connected_by (str | Unset):
            key_rejected_at (datetime.datetime | Unset):
     """

    account: EarnAccount
    create_key_url: str
    dashboard_url: str
    demo: bool
    emails_url: str
    granted: int
    health: EarnConnectionHealth
    id: str
    key_hint: str
    keys_url: str
    mode: EarnConnectionMode
    permissions: list[EarnPermissionCheck]
    portal_set: bool
    provider: EarnConnectionProvider
    provider_name: str
    status: EarnConnectionStatus
    step: EarnConnectionStep
    webhook: EarnWebhook
    webhooks_url: str
    checked_at: datetime.datetime | Unset = UNSET
    connected_at: datetime.datetime | Unset = UNSET
    connected_by: str | Unset = UNSET
    key_rejected_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.earn_account import EarnAccount # noqa: PLC0415
        from ..models.earn_permission_check import EarnPermissionCheck # noqa: PLC0415
        from ..models.earn_webhook import EarnWebhook # noqa: PLC0415
        account = self.account.to_dict()

        create_key_url = self.create_key_url

        dashboard_url = self.dashboard_url

        demo = self.demo

        emails_url = self.emails_url

        granted = self.granted

        health = self.health.value

        id = self.id

        key_hint = self.key_hint

        keys_url = self.keys_url

        mode = self.mode.value

        permissions = []
        for permissions_item_data in self.permissions:
            permissions_item = permissions_item_data.to_dict()
            permissions.append(permissions_item)



        portal_set = self.portal_set

        provider = self.provider.value

        provider_name = self.provider_name

        status = self.status.value

        step = self.step.value

        webhook = self.webhook.to_dict()

        webhooks_url = self.webhooks_url

        checked_at: str | Unset = UNSET
        if not isinstance(self.checked_at, Unset):
            checked_at = self.checked_at.isoformat()

        connected_at: str | Unset = UNSET
        if not isinstance(self.connected_at, Unset):
            connected_at = self.connected_at.isoformat()

        connected_by = self.connected_by

        key_rejected_at: str | Unset = UNSET
        if not isinstance(self.key_rejected_at, Unset):
            key_rejected_at = self.key_rejected_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "account": account,
            "createKeyUrl": create_key_url,
            "dashboardUrl": dashboard_url,
            "demo": demo,
            "emailsUrl": emails_url,
            "granted": granted,
            "health": health,
            "id": id,
            "keyHint": key_hint,
            "keysUrl": keys_url,
            "mode": mode,
            "permissions": permissions,
            "portalSet": portal_set,
            "provider": provider,
            "providerName": provider_name,
            "status": status,
            "step": step,
            "webhook": webhook,
            "webhooksUrl": webhooks_url,
        })
        if checked_at is not UNSET:
            field_dict["checkedAt"] = checked_at
        if connected_at is not UNSET:
            field_dict["connectedAt"] = connected_at
        if connected_by is not UNSET:
            field_dict["connectedBy"] = connected_by
        if key_rejected_at is not UNSET:
            field_dict["keyRejectedAt"] = key_rejected_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.earn_account import EarnAccount # noqa: PLC0415
        from ..models.earn_permission_check import EarnPermissionCheck # noqa: PLC0415
        from ..models.earn_webhook import EarnWebhook # noqa: PLC0415
        d = dict(src_dict)
        account = EarnAccount.from_dict(d.pop("account"))




        create_key_url = d.pop("createKeyUrl")

        dashboard_url = d.pop("dashboardUrl")

        demo = d.pop("demo")

        emails_url = d.pop("emailsUrl")

        granted = d.pop("granted")

        health = EarnConnectionHealth(d.pop("health"))




        id = d.pop("id")

        key_hint = d.pop("keyHint")

        keys_url = d.pop("keysUrl")

        mode = EarnConnectionMode(d.pop("mode"))




        permissions = []
        _permissions = d.pop("permissions")
        for permissions_item_data in (_permissions):
            permissions_item = EarnPermissionCheck.from_dict(permissions_item_data)



            permissions.append(permissions_item)


        portal_set = d.pop("portalSet")

        provider = EarnConnectionProvider(d.pop("provider"))




        provider_name = d.pop("providerName")

        status = EarnConnectionStatus(d.pop("status"))




        step = EarnConnectionStep(d.pop("step"))




        webhook = EarnWebhook.from_dict(d.pop("webhook"))




        webhooks_url = d.pop("webhooksUrl")

        _checked_at = d.pop("checkedAt", UNSET)
        checked_at: datetime.datetime | Unset
        if isinstance(_checked_at,  Unset):
            checked_at = UNSET
        else:
            checked_at = datetime.datetime.fromisoformat(_checked_at)




        _connected_at = d.pop("connectedAt", UNSET)
        connected_at: datetime.datetime | Unset
        if isinstance(_connected_at,  Unset):
            connected_at = UNSET
        else:
            connected_at = datetime.datetime.fromisoformat(_connected_at)




        connected_by = d.pop("connectedBy", UNSET)

        _key_rejected_at = d.pop("keyRejectedAt", UNSET)
        key_rejected_at: datetime.datetime | Unset
        if isinstance(_key_rejected_at,  Unset):
            key_rejected_at = UNSET
        else:
            key_rejected_at = datetime.datetime.fromisoformat(_key_rejected_at)




        earn_connection = cls(
            account=account,
            create_key_url=create_key_url,
            dashboard_url=dashboard_url,
            demo=demo,
            emails_url=emails_url,
            granted=granted,
            health=health,
            id=id,
            key_hint=key_hint,
            keys_url=keys_url,
            mode=mode,
            permissions=permissions,
            portal_set=portal_set,
            provider=provider,
            provider_name=provider_name,
            status=status,
            step=step,
            webhook=webhook,
            webhooks_url=webhooks_url,
            checked_at=checked_at,
            connected_at=connected_at,
            connected_by=connected_by,
            key_rejected_at=key_rejected_at,
        )

        return earn_connection

