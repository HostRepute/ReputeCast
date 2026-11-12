from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.webhook_view_status import WebhookViewStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.webhook_managed_by import WebhookManagedBy





T = TypeVar("T", bound="WebhookView")



@_attrs_define
class WebhookView:
    """ 
        Attributes:
            batch_email (bool):
            created_at (datetime.datetime):
            description (str):
            enabled (bool):
            events (list[str]):
            id (str):
            name (str):
            queued (int): Deliveries waiting for an attempt or a retry
            status (WebhookViewStatus):
            total24h (int): Deliveries attempted in the last 24 hours
            url (str):
            failing_since (datetime.datetime | Unset):
            last_delivery_at (datetime.datetime | Unset):
            last_status (int | Unset):
            managed_by (WebhookManagedBy | Unset):
            paused_reason (str | Unset):
            previous_secret_until (datetime.datetime | Unset):
            secret_rotated_at (datetime.datetime | Unset): Set when the secret was replaced after creation
            success24h (float | Unset): Percent delivered in the last 24 hours
     """

    batch_email: bool
    created_at: datetime.datetime
    description: str
    enabled: bool
    events: list[str]
    id: str
    name: str
    queued: int
    status: WebhookViewStatus
    total24h: int
    url: str
    failing_since: datetime.datetime | Unset = UNSET
    last_delivery_at: datetime.datetime | Unset = UNSET
    last_status: int | Unset = UNSET
    managed_by: WebhookManagedBy | Unset = UNSET
    paused_reason: str | Unset = UNSET
    previous_secret_until: datetime.datetime | Unset = UNSET
    secret_rotated_at: datetime.datetime | Unset = UNSET
    success24h: float | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.webhook_managed_by import WebhookManagedBy # noqa: PLC0415
        batch_email = self.batch_email

        created_at = self.created_at.isoformat()

        description = self.description

        enabled = self.enabled

        events = self.events



        id = self.id

        name = self.name

        queued = self.queued

        status = self.status.value

        total24h = self.total24h

        url = self.url

        failing_since: str | Unset = UNSET
        if not isinstance(self.failing_since, Unset):
            failing_since = self.failing_since.isoformat()

        last_delivery_at: str | Unset = UNSET
        if not isinstance(self.last_delivery_at, Unset):
            last_delivery_at = self.last_delivery_at.isoformat()

        last_status = self.last_status

        managed_by: dict[str, Any] | Unset = UNSET
        if not isinstance(self.managed_by, Unset):
            managed_by = self.managed_by.to_dict()

        paused_reason = self.paused_reason

        previous_secret_until: str | Unset = UNSET
        if not isinstance(self.previous_secret_until, Unset):
            previous_secret_until = self.previous_secret_until.isoformat()

        secret_rotated_at: str | Unset = UNSET
        if not isinstance(self.secret_rotated_at, Unset):
            secret_rotated_at = self.secret_rotated_at.isoformat()

        success24h = self.success24h


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "batchEmail": batch_email,
            "createdAt": created_at,
            "description": description,
            "enabled": enabled,
            "events": events,
            "id": id,
            "name": name,
            "queued": queued,
            "status": status,
            "total24h": total24h,
            "url": url,
        })
        if failing_since is not UNSET:
            field_dict["failingSince"] = failing_since
        if last_delivery_at is not UNSET:
            field_dict["lastDeliveryAt"] = last_delivery_at
        if last_status is not UNSET:
            field_dict["lastStatus"] = last_status
        if managed_by is not UNSET:
            field_dict["managedBy"] = managed_by
        if paused_reason is not UNSET:
            field_dict["pausedReason"] = paused_reason
        if previous_secret_until is not UNSET:
            field_dict["previousSecretUntil"] = previous_secret_until
        if secret_rotated_at is not UNSET:
            field_dict["secretRotatedAt"] = secret_rotated_at
        if success24h is not UNSET:
            field_dict["success24h"] = success24h

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.webhook_managed_by import WebhookManagedBy # noqa: PLC0415
        d = dict(src_dict)
        batch_email = d.pop("batchEmail")

        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))




        description = d.pop("description")

        enabled = d.pop("enabled")

        events = cast(list[str], d.pop("events"))


        id = d.pop("id")

        name = d.pop("name")

        queued = d.pop("queued")

        status = WebhookViewStatus(d.pop("status"))




        total24h = d.pop("total24h")

        url = d.pop("url")

        _failing_since = d.pop("failingSince", UNSET)
        failing_since: datetime.datetime | Unset
        if isinstance(_failing_since,  Unset):
            failing_since = UNSET
        else:
            failing_since = datetime.datetime.fromisoformat(_failing_since)




        _last_delivery_at = d.pop("lastDeliveryAt", UNSET)
        last_delivery_at: datetime.datetime | Unset
        if isinstance(_last_delivery_at,  Unset):
            last_delivery_at = UNSET
        else:
            last_delivery_at = datetime.datetime.fromisoformat(_last_delivery_at)




        last_status = d.pop("lastStatus", UNSET)

        _managed_by = d.pop("managedBy", UNSET)
        managed_by: WebhookManagedBy | Unset
        if isinstance(_managed_by,  Unset):
            managed_by = UNSET
        else:
            managed_by = WebhookManagedBy.from_dict(_managed_by)




        paused_reason = d.pop("pausedReason", UNSET)

        _previous_secret_until = d.pop("previousSecretUntil", UNSET)
        previous_secret_until: datetime.datetime | Unset
        if isinstance(_previous_secret_until,  Unset):
            previous_secret_until = UNSET
        else:
            previous_secret_until = datetime.datetime.fromisoformat(_previous_secret_until)




        _secret_rotated_at = d.pop("secretRotatedAt", UNSET)
        secret_rotated_at: datetime.datetime | Unset
        if isinstance(_secret_rotated_at,  Unset):
            secret_rotated_at = UNSET
        else:
            secret_rotated_at = datetime.datetime.fromisoformat(_secret_rotated_at)




        success24h = d.pop("success24h", UNSET)

        webhook_view = cls(
            batch_email=batch_email,
            created_at=created_at,
            description=description,
            enabled=enabled,
            events=events,
            id=id,
            name=name,
            queued=queued,
            status=status,
            total24h=total24h,
            url=url,
            failing_since=failing_since,
            last_delivery_at=last_delivery_at,
            last_status=last_status,
            managed_by=managed_by,
            paused_reason=paused_reason,
            previous_secret_until=previous_secret_until,
            secret_rotated_at=secret_rotated_at,
            success24h=success24h,
        )

        return webhook_view

