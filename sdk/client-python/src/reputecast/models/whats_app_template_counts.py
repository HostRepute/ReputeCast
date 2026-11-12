from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="WhatsAppTemplateCounts")



@_attrs_define
class WhatsAppTemplateCounts:
    """ 
        Attributes:
            approved (int):
            other (int):
            pending (int):
            rejected (int):
     """

    approved: int
    other: int
    pending: int
    rejected: int





    def to_dict(self) -> dict[str, Any]:
        approved = self.approved

        other = self.other

        pending = self.pending

        rejected = self.rejected


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "approved": approved,
            "other": other,
            "pending": pending,
            "rejected": rejected,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        approved = d.pop("approved")

        other = d.pop("other")

        pending = d.pop("pending")

        rejected = d.pop("rejected")

        whats_app_template_counts = cls(
            approved=approved,
            other=other,
            pending=pending,
            rejected=rejected,
        )

        return whats_app_template_counts

