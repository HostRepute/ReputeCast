from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ApprovalCounts")



@_attrs_define
class ApprovalCounts:
    """ 
        Attributes:
            all_ (int):
            approved (int):
            changes_requested (int):
            invalidated (int):
            pending (int):
            rejected (int):
     """

    all_: int
    approved: int
    changes_requested: int
    invalidated: int
    pending: int
    rejected: int





    def to_dict(self) -> dict[str, Any]:
        all_ = self.all_

        approved = self.approved

        changes_requested = self.changes_requested

        invalidated = self.invalidated

        pending = self.pending

        rejected = self.rejected


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "all": all_,
            "approved": approved,
            "changesRequested": changes_requested,
            "invalidated": invalidated,
            "pending": pending,
            "rejected": rejected,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        all_ = d.pop("all")

        approved = d.pop("approved")

        changes_requested = d.pop("changesRequested")

        invalidated = d.pop("invalidated")

        pending = d.pop("pending")

        rejected = d.pop("rejected")

        approval_counts = cls(
            all_=all_,
            approved=approved,
            changes_requested=changes_requested,
            invalidated=invalidated,
            pending=pending,
            rejected=rejected,
        )

        return approval_counts

