from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="CampaignBulkSkip")



@_attrs_define
class CampaignBulkSkip:
    """ 
        Attributes:
            id (str):
            name (str):
            reason (str):
     """

    id: str
    name: str
    reason: str





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        name = self.name

        reason = self.reason


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "name": name,
            "reason": reason,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        name = d.pop("name")

        reason = d.pop("reason")

        campaign_bulk_skip = cls(
            id=id,
            name=name,
            reason=reason,
        )

        return campaign_bulk_skip

