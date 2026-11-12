from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.live_mailbox_complaint_reason import LiveMailboxComplaintReason
from ..models.live_mailbox_key import LiveMailboxKey
from ..models.live_mailbox_status import LiveMailboxStatus
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="LiveMailbox")



@_attrs_define
class LiveMailbox:
    """ 
        Attributes:
            bounced (int):
            complained (int):
            deferred (int):
            delivered (int):
            held (bool):
            key (LiveMailboxKey):
            name (str):
            opened (int):
            remaining (int):
            retrying (int):
            sent (int):
            status (LiveMailboxStatus):
            bounce_rate (float | Unset):
            complaint_rate (float | Unset):
            complaint_reason (LiveMailboxComplaintReason | Unset):
            deferred_rate (float | Unset):
            delivered_rate (float | Unset):
            done_at (datetime.datetime | Unset):
            held_since (datetime.datetime | Unset):
            held_until (datetime.datetime | Unset):
            open_rate (float | Unset):
            throttle (float | Unset): Messages per second; absent = the provider's full rate
     """

    bounced: int
    complained: int
    deferred: int
    delivered: int
    held: bool
    key: LiveMailboxKey
    name: str
    opened: int
    remaining: int
    retrying: int
    sent: int
    status: LiveMailboxStatus
    bounce_rate: float | Unset = UNSET
    complaint_rate: float | Unset = UNSET
    complaint_reason: LiveMailboxComplaintReason | Unset = UNSET
    deferred_rate: float | Unset = UNSET
    delivered_rate: float | Unset = UNSET
    done_at: datetime.datetime | Unset = UNSET
    held_since: datetime.datetime | Unset = UNSET
    held_until: datetime.datetime | Unset = UNSET
    open_rate: float | Unset = UNSET
    throttle: float | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        bounced = self.bounced

        complained = self.complained

        deferred = self.deferred

        delivered = self.delivered

        held = self.held

        key = self.key.value

        name = self.name

        opened = self.opened

        remaining = self.remaining

        retrying = self.retrying

        sent = self.sent

        status = self.status.value

        bounce_rate = self.bounce_rate

        complaint_rate = self.complaint_rate

        complaint_reason: str | Unset = UNSET
        if not isinstance(self.complaint_reason, Unset):
            complaint_reason = self.complaint_reason.value


        deferred_rate = self.deferred_rate

        delivered_rate = self.delivered_rate

        done_at: str | Unset = UNSET
        if not isinstance(self.done_at, Unset):
            done_at = self.done_at.isoformat()

        held_since: str | Unset = UNSET
        if not isinstance(self.held_since, Unset):
            held_since = self.held_since.isoformat()

        held_until: str | Unset = UNSET
        if not isinstance(self.held_until, Unset):
            held_until = self.held_until.isoformat()

        open_rate = self.open_rate

        throttle = self.throttle


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "bounced": bounced,
            "complained": complained,
            "deferred": deferred,
            "delivered": delivered,
            "held": held,
            "key": key,
            "name": name,
            "opened": opened,
            "remaining": remaining,
            "retrying": retrying,
            "sent": sent,
            "status": status,
        })
        if bounce_rate is not UNSET:
            field_dict["bounceRate"] = bounce_rate
        if complaint_rate is not UNSET:
            field_dict["complaintRate"] = complaint_rate
        if complaint_reason is not UNSET:
            field_dict["complaintReason"] = complaint_reason
        if deferred_rate is not UNSET:
            field_dict["deferredRate"] = deferred_rate
        if delivered_rate is not UNSET:
            field_dict["deliveredRate"] = delivered_rate
        if done_at is not UNSET:
            field_dict["doneAt"] = done_at
        if held_since is not UNSET:
            field_dict["heldSince"] = held_since
        if held_until is not UNSET:
            field_dict["heldUntil"] = held_until
        if open_rate is not UNSET:
            field_dict["openRate"] = open_rate
        if throttle is not UNSET:
            field_dict["throttle"] = throttle

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        bounced = d.pop("bounced")

        complained = d.pop("complained")

        deferred = d.pop("deferred")

        delivered = d.pop("delivered")

        held = d.pop("held")

        key = LiveMailboxKey(d.pop("key"))




        name = d.pop("name")

        opened = d.pop("opened")

        remaining = d.pop("remaining")

        retrying = d.pop("retrying")

        sent = d.pop("sent")

        status = LiveMailboxStatus(d.pop("status"))




        bounce_rate = d.pop("bounceRate", UNSET)

        complaint_rate = d.pop("complaintRate", UNSET)

        _complaint_reason = d.pop("complaintReason", UNSET)
        complaint_reason: LiveMailboxComplaintReason | Unset
        if isinstance(_complaint_reason,  Unset):
            complaint_reason = UNSET
        else:
            complaint_reason = LiveMailboxComplaintReason(_complaint_reason)




        deferred_rate = d.pop("deferredRate", UNSET)

        delivered_rate = d.pop("deliveredRate", UNSET)

        _done_at = d.pop("doneAt", UNSET)
        done_at: datetime.datetime | Unset
        if isinstance(_done_at,  Unset):
            done_at = UNSET
        else:
            done_at = datetime.datetime.fromisoformat(_done_at)




        _held_since = d.pop("heldSince", UNSET)
        held_since: datetime.datetime | Unset
        if isinstance(_held_since,  Unset):
            held_since = UNSET
        else:
            held_since = datetime.datetime.fromisoformat(_held_since)




        _held_until = d.pop("heldUntil", UNSET)
        held_until: datetime.datetime | Unset
        if isinstance(_held_until,  Unset):
            held_until = UNSET
        else:
            held_until = datetime.datetime.fromisoformat(_held_until)




        open_rate = d.pop("openRate", UNSET)

        throttle = d.pop("throttle", UNSET)

        live_mailbox = cls(
            bounced=bounced,
            complained=complained,
            deferred=deferred,
            delivered=delivered,
            held=held,
            key=key,
            name=name,
            opened=opened,
            remaining=remaining,
            retrying=retrying,
            sent=sent,
            status=status,
            bounce_rate=bounce_rate,
            complaint_rate=complaint_rate,
            complaint_reason=complaint_reason,
            deferred_rate=deferred_rate,
            delivered_rate=delivered_rate,
            done_at=done_at,
            held_since=held_since,
            held_until=held_until,
            open_rate=open_rate,
            throttle=throttle,
        )

        return live_mailbox

