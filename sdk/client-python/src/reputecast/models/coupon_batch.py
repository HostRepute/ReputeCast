from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.coupon_batch_kind import CouponBatchKind
from ..models.coupon_batch_reason import CouponBatchReason
from ..models.coupon_batch_status import CouponBatchStatus
from ..types import UNSET, Unset
from typing import cast
from uuid import UUID
import datetime






T = TypeVar("T", bound="CouponBatch")



@_attrs_define
class CouponBatch:
    """ 
        Attributes:
            created (int):
            created_at (datetime.datetime):
            failed (int):
            id (UUID):
            kind (CouponBatchKind):
            reason (CouponBatchReason):
            requested (int):
            status (CouponBatchStatus):
            automation (str | Unset):
            by (str | Unset): Who asked, by name
            error (str | Unset):
            finished_at (datetime.datetime | Unset):
            waiting_until (datetime.datetime | Unset):
     """

    created: int
    created_at: datetime.datetime
    failed: int
    id: UUID
    kind: CouponBatchKind
    reason: CouponBatchReason
    requested: int
    status: CouponBatchStatus
    automation: str | Unset = UNSET
    by: str | Unset = UNSET
    error: str | Unset = UNSET
    finished_at: datetime.datetime | Unset = UNSET
    waiting_until: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        created = self.created

        created_at = self.created_at.isoformat()

        failed = self.failed

        id = str(self.id)

        kind = self.kind.value

        reason = self.reason.value

        requested = self.requested

        status = self.status.value

        automation = self.automation

        by = self.by

        error = self.error

        finished_at: str | Unset = UNSET
        if not isinstance(self.finished_at, Unset):
            finished_at = self.finished_at.isoformat()

        waiting_until: str | Unset = UNSET
        if not isinstance(self.waiting_until, Unset):
            waiting_until = self.waiting_until.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "created": created,
            "createdAt": created_at,
            "failed": failed,
            "id": id,
            "kind": kind,
            "reason": reason,
            "requested": requested,
            "status": status,
        })
        if automation is not UNSET:
            field_dict["automation"] = automation
        if by is not UNSET:
            field_dict["by"] = by
        if error is not UNSET:
            field_dict["error"] = error
        if finished_at is not UNSET:
            field_dict["finishedAt"] = finished_at
        if waiting_until is not UNSET:
            field_dict["waitingUntil"] = waiting_until

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        created = d.pop("created")

        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))




        failed = d.pop("failed")

        id = UUID(d.pop("id"))




        kind = CouponBatchKind(d.pop("kind"))




        reason = CouponBatchReason(d.pop("reason"))




        requested = d.pop("requested")

        status = CouponBatchStatus(d.pop("status"))




        automation = d.pop("automation", UNSET)

        by = d.pop("by", UNSET)

        error = d.pop("error", UNSET)

        _finished_at = d.pop("finishedAt", UNSET)
        finished_at: datetime.datetime | Unset
        if isinstance(_finished_at,  Unset):
            finished_at = UNSET
        else:
            finished_at = datetime.datetime.fromisoformat(_finished_at)




        _waiting_until = d.pop("waitingUntil", UNSET)
        waiting_until: datetime.datetime | Unset
        if isinstance(_waiting_until,  Unset):
            waiting_until = UNSET
        else:
            waiting_until = datetime.datetime.fromisoformat(_waiting_until)




        coupon_batch = cls(
            created=created,
            created_at=created_at,
            failed=failed,
            id=id,
            kind=kind,
            reason=reason,
            requested=requested,
            status=status,
            automation=automation,
            by=by,
            error=error,
            finished_at=finished_at,
            waiting_until=waiting_until,
        )

        return coupon_batch

