from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.routing_vet_check_key import RoutingVetCheckKey
from ..models.routing_vet_check_period import RoutingVetCheckPeriod
from ..models.routing_vet_check_result import RoutingVetCheckResult
from ..types import UNSET, Unset






T = TypeVar("T", bound="RoutingVetCheck")



@_attrs_define
class RoutingVetCheck:
    """ 
        Attributes:
            key (RoutingVetCheckKey):
            result (RoutingVetCheckResult):
            left (int | Unset):
            period (RoutingVetCheckPeriod | Unset):
     """

    key: RoutingVetCheckKey
    result: RoutingVetCheckResult
    left: int | Unset = UNSET
    period: RoutingVetCheckPeriod | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        key = self.key.value

        result = self.result.value

        left = self.left

        period: str | Unset = UNSET
        if not isinstance(self.period, Unset):
            period = self.period.value



        field_dict: dict[str, Any] = {}

        field_dict.update({
            "key": key,
            "result": result,
        })
        if left is not UNSET:
            field_dict["left"] = left
        if period is not UNSET:
            field_dict["period"] = period

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        key = RoutingVetCheckKey(d.pop("key"))




        result = RoutingVetCheckResult(d.pop("result"))




        left = d.pop("left", UNSET)

        _period = d.pop("period", UNSET)
        period: RoutingVetCheckPeriod | Unset
        if isinstance(_period,  Unset):
            period = UNSET
        else:
            period = RoutingVetCheckPeriod(_period)




        routing_vet_check = cls(
            key=key,
            result=result,
            left=left,
            period=period,
        )

        return routing_vet_check

