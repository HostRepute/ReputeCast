from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.badge_capability import BadgeCapability
from ..models.badge_support import BadgeSupport
from ..types import UNSET, Unset






T = TypeVar("T", bound="Badge")



@_attrs_define
class Badge:
    """ 
        Attributes:
            capability (BadgeCapability):
            support (BadgeSupport):
            note (str | Unset):
     """

    capability: BadgeCapability
    support: BadgeSupport
    note: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        capability = self.capability.value

        support = self.support.value

        note = self.note


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "capability": capability,
            "support": support,
        })
        if note is not UNSET:
            field_dict["note"] = note

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        capability = BadgeCapability(d.pop("capability"))




        support = BadgeSupport(d.pop("support"))




        note = d.pop("note", UNSET)

        badge = cls(
            capability=capability,
            support=support,
            note=note,
        )

        return badge

