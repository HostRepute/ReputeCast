from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from uuid import UUID






T = TypeVar("T", bound="RoutingSplit")



@_attrs_define
class RoutingSplit:
    """ 
        Attributes:
            provider_id (UUID):
            weight (int):
     """

    provider_id: UUID
    weight: int





    def to_dict(self) -> dict[str, Any]:
        provider_id = str(self.provider_id)

        weight = self.weight


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "providerId": provider_id,
            "weight": weight,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        provider_id = UUID(d.pop("providerId"))




        weight = d.pop("weight")

        routing_split = cls(
            provider_id=provider_id,
            weight=weight,
        )

        return routing_split

