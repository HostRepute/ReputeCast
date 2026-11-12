from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.growth_source import GrowthSource





T = TypeVar("T", bound="GrowthSeries")



@_attrs_define
class GrowthSeries:
    """ 
        Attributes:
            joined (int):
            key (str):
            left (int):
            other (bool):
            source (GrowthSource):
     """

    joined: int
    key: str
    left: int
    other: bool
    source: GrowthSource





    def to_dict(self) -> dict[str, Any]:
        from ..models.growth_source import GrowthSource # noqa: PLC0415
        joined = self.joined

        key = self.key

        left = self.left

        other = self.other

        source = self.source.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "joined": joined,
            "key": key,
            "left": left,
            "other": other,
            "source": source,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.growth_source import GrowthSource # noqa: PLC0415
        d = dict(src_dict)
        joined = d.pop("joined")

        key = d.pop("key")

        left = d.pop("left")

        other = d.pop("other")

        source = GrowthSource.from_dict(d.pop("source"))




        growth_series = cls(
            joined=joined,
            key=key,
            left=left,
            other=other,
            source=source,
        )

        return growth_series

