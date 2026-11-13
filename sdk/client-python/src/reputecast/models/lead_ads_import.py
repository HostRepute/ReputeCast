from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.lead_ads_import_state import LeadAdsImportState
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="LeadAdsImport")



@_attrs_define
class LeadAdsImport:
    """ 
        Attributes:
            count (int): Leads read so far, or in the finished import
            state (LeadAdsImportState):
            at (datetime.datetime | Unset): When the last import finished
     """

    count: int
    state: LeadAdsImportState
    at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        count = self.count

        state = self.state.value

        at: str | Unset = UNSET
        if not isinstance(self.at, Unset):
            at = self.at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "count": count,
            "state": state,
        })
        if at is not UNSET:
            field_dict["at"] = at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        count = d.pop("count")

        state = LeadAdsImportState(d.pop("state"))




        _at = d.pop("at", UNSET)
        at: datetime.datetime | Unset
        if isinstance(_at,  Unset):
            at = UNSET
        else:
            at = datetime.datetime.fromisoformat(_at)




        lead_ads_import = cls(
            count=count,
            state=state,
            at=at,
        )

        return lead_ads_import

