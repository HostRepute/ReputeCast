from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="TransactionalLogCounts")



@_attrs_define
class TransactionalLogCounts:
    """ 
        Attributes:
            all_ (int):
            bounced (int):
            cancelled (int):
            complained (int):
            deferred (int):
            delivered (int):
            failed (int):
            queued (int):
            rejected (int):
            scheduled (int):
            sent (int):
            test (int):
     """

    all_: int
    bounced: int
    cancelled: int
    complained: int
    deferred: int
    delivered: int
    failed: int
    queued: int
    rejected: int
    scheduled: int
    sent: int
    test: int





    def to_dict(self) -> dict[str, Any]:
        all_ = self.all_

        bounced = self.bounced

        cancelled = self.cancelled

        complained = self.complained

        deferred = self.deferred

        delivered = self.delivered

        failed = self.failed

        queued = self.queued

        rejected = self.rejected

        scheduled = self.scheduled

        sent = self.sent

        test = self.test


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "all": all_,
            "bounced": bounced,
            "cancelled": cancelled,
            "complained": complained,
            "deferred": deferred,
            "delivered": delivered,
            "failed": failed,
            "queued": queued,
            "rejected": rejected,
            "scheduled": scheduled,
            "sent": sent,
            "test": test,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        all_ = d.pop("all")

        bounced = d.pop("bounced")

        cancelled = d.pop("cancelled")

        complained = d.pop("complained")

        deferred = d.pop("deferred")

        delivered = d.pop("delivered")

        failed = d.pop("failed")

        queued = d.pop("queued")

        rejected = d.pop("rejected")

        scheduled = d.pop("scheduled")

        sent = d.pop("sent")

        test = d.pop("test")

        transactional_log_counts = cls(
            all_=all_,
            bounced=bounced,
            cancelled=cancelled,
            complained=complained,
            deferred=deferred,
            delivered=delivered,
            failed=failed,
            queued=queued,
            rejected=rejected,
            scheduled=scheduled,
            sent=sent,
            test=test,
        )

        return transactional_log_counts

