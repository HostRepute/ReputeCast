from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.routing_volume_per import RoutingVolumePer
from ..types import UNSET, Unset






T = TypeVar("T", bound="RoutingVolume")



@_attrs_define
class RoutingVolume:
    """ 
        Attributes:
            count (int):
            per (RoutingVolumePer):
            largest (int | Unset):
            smallest (int | Unset):
     """

    count: int
    per: RoutingVolumePer
    largest: int | Unset = UNSET
    smallest: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        count = self.count

        per = self.per.value

        largest = self.largest

        smallest = self.smallest


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "count": count,
            "per": per,
        })
        if largest is not UNSET:
            field_dict["largest"] = largest
        if smallest is not UNSET:
            field_dict["smallest"] = smallest

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        count = d.pop("count")

        per = RoutingVolumePer(d.pop("per"))




        largest = d.pop("largest", UNSET)

        smallest = d.pop("smallest", UNSET)

        routing_volume = cls(
            count=count,
            per=per,
            largest=largest,
            smallest=smallest,
        )

        return routing_volume

