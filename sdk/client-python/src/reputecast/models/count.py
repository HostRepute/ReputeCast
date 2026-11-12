from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.count_encoding import CountEncoding
from typing import cast

if TYPE_CHECKING:
  from ..models.outside import Outside





T = TypeVar("T", bound="Count")



@_attrs_define
class Count:
    """ 
        Attributes:
            capacity (int):
            characters (int):
            encoding (CountEncoding):
            fill (list[int]):
            left (int):
            outside (list[Outside]):
            segments (int):
            units (int):
     """

    capacity: int
    characters: int
    encoding: CountEncoding
    fill: list[int]
    left: int
    outside: list[Outside]
    segments: int
    units: int





    def to_dict(self) -> dict[str, Any]:
        from ..models.outside import Outside # noqa: PLC0415
        capacity = self.capacity

        characters = self.characters

        encoding = self.encoding.value

        fill = self.fill



        left = self.left

        outside = []
        for outside_item_data in self.outside:
            outside_item = outside_item_data.to_dict()
            outside.append(outside_item)



        segments = self.segments

        units = self.units


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "capacity": capacity,
            "characters": characters,
            "encoding": encoding,
            "fill": fill,
            "left": left,
            "outside": outside,
            "segments": segments,
            "units": units,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.outside import Outside # noqa: PLC0415
        d = dict(src_dict)
        capacity = d.pop("capacity")

        characters = d.pop("characters")

        encoding = CountEncoding(d.pop("encoding"))




        fill = cast(list[int], d.pop("fill"))


        left = d.pop("left")

        outside = []
        _outside = d.pop("outside")
        for outside_item_data in (_outside):
            outside_item = Outside.from_dict(outside_item_data)



            outside.append(outside_item)


        segments = d.pop("segments")

        units = d.pop("units")

        count = cls(
            capacity=capacity,
            characters=characters,
            encoding=encoding,
            fill=fill,
            left=left,
            outside=outside,
            segments=segments,
            units=units,
        )

        return count

