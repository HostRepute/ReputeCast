from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.lead_ads_connection_item_status import LeadAdsConnectionItemStatus
from typing import cast
import datetime






T = TypeVar("T", bound="LeadAdsConnectionItem")



@_attrs_define
class LeadAdsConnectionItem:
    """ 
        Attributes:
            created_at (datetime.datetime):
            id (str):
            name (str):  Example: Acme Marketing.
            status (LeadAdsConnectionItemStatus):
     """

    created_at: datetime.datetime
    id: str
    name: str
    status: LeadAdsConnectionItemStatus





    def to_dict(self) -> dict[str, Any]:
        created_at = self.created_at.isoformat()

        id = self.id

        name = self.name

        status = self.status.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "createdAt": created_at,
            "id": id,
            "name": name,
            "status": status,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))




        id = d.pop("id")

        name = d.pop("name")

        status = LeadAdsConnectionItemStatus(d.pop("status"))




        lead_ads_connection_item = cls(
            created_at=created_at,
            id=id,
            name=name,
            status=status,
        )

        return lead_ads_connection_item

