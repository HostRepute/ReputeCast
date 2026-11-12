from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="RoutingShared")



@_attrs_define
class RoutingShared:
    """ 
        Attributes:
            pushed (int):
            suppressions (int):
            synced_at (datetime.datetime | Unset):
     """

    pushed: int
    suppressions: int
    synced_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        pushed = self.pushed

        suppressions = self.suppressions

        synced_at: str | Unset = UNSET
        if not isinstance(self.synced_at, Unset):
            synced_at = self.synced_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "pushed": pushed,
            "suppressions": suppressions,
        })
        if synced_at is not UNSET:
            field_dict["syncedAt"] = synced_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        pushed = d.pop("pushed")

        suppressions = d.pop("suppressions")

        _synced_at = d.pop("syncedAt", UNSET)
        synced_at: datetime.datetime | Unset
        if isinstance(_synced_at,  Unset):
            synced_at = UNSET
        else:
            synced_at = datetime.datetime.fromisoformat(_synced_at)




        routing_shared = cls(
            pushed=pushed,
            suppressions=suppressions,
            synced_at=synced_at,
        )

        return routing_shared

