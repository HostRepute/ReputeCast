from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.integration_managed_hook_status import IntegrationManagedHookStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.integration_attempt import IntegrationAttempt





T = TypeVar("T", bound="IntegrationManagedHook")



@_attrs_define
class IntegrationManagedHook:
    """ 
        Attributes:
            created_at (datetime.datetime):
            dead (int):
            delivered24h (int):
            endpoint (str): Masked: https://n8n.acme.io/webhook/7c1e…/reputecast
            event (str):
            id (str):
            queued (int):
            retrying (int):
            status (IntegrationManagedHookStatus):
            total24h (int):
            failing_since (datetime.datetime | Unset):
            last_attempt (IntegrationAttempt | Unset):
            last_status (int | Unset):
            next_at (datetime.datetime | Unset):
            recovered_at (datetime.datetime | Unset):
            status_text (str | Unset):
            workflow (str | Unset):
     """

    created_at: datetime.datetime
    dead: int
    delivered24h: int
    endpoint: str
    event: str
    id: str
    queued: int
    retrying: int
    status: IntegrationManagedHookStatus
    total24h: int
    failing_since: datetime.datetime | Unset = UNSET
    last_attempt: IntegrationAttempt | Unset = UNSET
    last_status: int | Unset = UNSET
    next_at: datetime.datetime | Unset = UNSET
    recovered_at: datetime.datetime | Unset = UNSET
    status_text: str | Unset = UNSET
    workflow: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.integration_attempt import IntegrationAttempt # noqa: PLC0415
        created_at = self.created_at.isoformat()

        dead = self.dead

        delivered24h = self.delivered24h

        endpoint = self.endpoint

        event = self.event

        id = self.id

        queued = self.queued

        retrying = self.retrying

        status = self.status.value

        total24h = self.total24h

        failing_since: str | Unset = UNSET
        if not isinstance(self.failing_since, Unset):
            failing_since = self.failing_since.isoformat()

        last_attempt: dict[str, Any] | Unset = UNSET
        if not isinstance(self.last_attempt, Unset):
            last_attempt = self.last_attempt.to_dict()

        last_status = self.last_status

        next_at: str | Unset = UNSET
        if not isinstance(self.next_at, Unset):
            next_at = self.next_at.isoformat()

        recovered_at: str | Unset = UNSET
        if not isinstance(self.recovered_at, Unset):
            recovered_at = self.recovered_at.isoformat()

        status_text = self.status_text

        workflow = self.workflow


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "createdAt": created_at,
            "dead": dead,
            "delivered24h": delivered24h,
            "endpoint": endpoint,
            "event": event,
            "id": id,
            "queued": queued,
            "retrying": retrying,
            "status": status,
            "total24h": total24h,
        })
        if failing_since is not UNSET:
            field_dict["failingSince"] = failing_since
        if last_attempt is not UNSET:
            field_dict["lastAttempt"] = last_attempt
        if last_status is not UNSET:
            field_dict["lastStatus"] = last_status
        if next_at is not UNSET:
            field_dict["nextAt"] = next_at
        if recovered_at is not UNSET:
            field_dict["recoveredAt"] = recovered_at
        if status_text is not UNSET:
            field_dict["statusText"] = status_text
        if workflow is not UNSET:
            field_dict["workflow"] = workflow

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.integration_attempt import IntegrationAttempt # noqa: PLC0415
        d = dict(src_dict)
        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))




        dead = d.pop("dead")

        delivered24h = d.pop("delivered24h")

        endpoint = d.pop("endpoint")

        event = d.pop("event")

        id = d.pop("id")

        queued = d.pop("queued")

        retrying = d.pop("retrying")

        status = IntegrationManagedHookStatus(d.pop("status"))




        total24h = d.pop("total24h")

        _failing_since = d.pop("failingSince", UNSET)
        failing_since: datetime.datetime | Unset
        if isinstance(_failing_since,  Unset):
            failing_since = UNSET
        else:
            failing_since = datetime.datetime.fromisoformat(_failing_since)




        _last_attempt = d.pop("lastAttempt", UNSET)
        last_attempt: IntegrationAttempt | Unset
        if isinstance(_last_attempt,  Unset):
            last_attempt = UNSET
        else:
            last_attempt = IntegrationAttempt.from_dict(_last_attempt)




        last_status = d.pop("lastStatus", UNSET)

        _next_at = d.pop("nextAt", UNSET)
        next_at: datetime.datetime | Unset
        if isinstance(_next_at,  Unset):
            next_at = UNSET
        else:
            next_at = datetime.datetime.fromisoformat(_next_at)




        _recovered_at = d.pop("recoveredAt", UNSET)
        recovered_at: datetime.datetime | Unset
        if isinstance(_recovered_at,  Unset):
            recovered_at = UNSET
        else:
            recovered_at = datetime.datetime.fromisoformat(_recovered_at)




        status_text = d.pop("statusText", UNSET)

        workflow = d.pop("workflow", UNSET)

        integration_managed_hook = cls(
            created_at=created_at,
            dead=dead,
            delivered24h=delivered24h,
            endpoint=endpoint,
            event=event,
            id=id,
            queued=queued,
            retrying=retrying,
            status=status,
            total24h=total24h,
            failing_since=failing_since,
            last_attempt=last_attempt,
            last_status=last_status,
            next_at=next_at,
            recovered_at=recovered_at,
            status_text=status_text,
            workflow=workflow,
        )

        return integration_managed_hook

