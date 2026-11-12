from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.growth_retention_state import GrowthRetentionState
from ..types import UNSET, Unset






T = TypeVar("T", bound="GrowthRetention")



@_attrs_define
class GrowthRetention:
    """ 
        Attributes:
            state (GrowthRetentionState):
            complete_on (str | Unset):
            from_ (str | Unset):
            rate (float | Unset):
     """

    state: GrowthRetentionState
    complete_on: str | Unset = UNSET
    from_: str | Unset = UNSET
    rate: float | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        state = self.state.value

        complete_on = self.complete_on

        from_ = self.from_

        rate = self.rate


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "state": state,
        })
        if complete_on is not UNSET:
            field_dict["completeOn"] = complete_on
        if from_ is not UNSET:
            field_dict["from"] = from_
        if rate is not UNSET:
            field_dict["rate"] = rate

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        state = GrowthRetentionState(d.pop("state"))




        complete_on = d.pop("completeOn", UNSET)

        from_ = d.pop("from", UNSET)

        rate = d.pop("rate", UNSET)

        growth_retention = cls(
            state=state,
            complete_on=complete_on,
            from_=from_,
            rate=rate,
        )

        return growth_retention

