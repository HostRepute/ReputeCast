from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="SendingDomainCounts")



@_attrs_define
class SendingDomainCounts:
    """ 
        Attributes:
            missing (int):
            pending (int):
            total (int):
            verified (int):
     """

    missing: int
    pending: int
    total: int
    verified: int





    def to_dict(self) -> dict[str, Any]:
        missing = self.missing

        pending = self.pending

        total = self.total

        verified = self.verified


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "missing": missing,
            "pending": pending,
            "total": total,
            "verified": verified,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        missing = d.pop("missing")

        pending = d.pop("pending")

        total = d.pop("total")

        verified = d.pop("verified")

        sending_domain_counts = cls(
            missing=missing,
            pending=pending,
            total=total,
            verified=verified,
        )

        return sending_domain_counts

