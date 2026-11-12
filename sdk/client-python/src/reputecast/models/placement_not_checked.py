from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.placement_not_checked_reason import PlacementNotCheckedReason






T = TypeVar("T", bound="PlacementNotChecked")



@_attrs_define
class PlacementNotChecked:
    """ 
        Attributes:
            email (str):
            name (str): The seed's mailbox provider (iCloud)
            provider (str):
            reason (PlacementNotCheckedReason):
     """

    email: str
    name: str
    provider: str
    reason: PlacementNotCheckedReason





    def to_dict(self) -> dict[str, Any]:
        email = self.email

        name = self.name

        provider = self.provider

        reason = self.reason.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "email": email,
            "name": name,
            "provider": provider,
            "reason": reason,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        email = d.pop("email")

        name = d.pop("name")

        provider = d.pop("provider")

        reason = PlacementNotCheckedReason(d.pop("reason"))




        placement_not_checked = cls(
            email=email,
            name=name,
            provider=provider,
            reason=reason,
        )

        return placement_not_checked

