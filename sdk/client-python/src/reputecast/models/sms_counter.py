from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.sms_counter_encoding import SmsCounterEncoding
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.outside import Outside
  from ..models.sms_field import SmsField
  from ..models.sms_longest import SmsLongest
  from ..models.sms_spread import SmsSpread





T = TypeVar("T", bound="SmsCounter")



@_attrs_define
class SmsCounter:
    """ 
        Attributes:
            base (int):
            capacity (int):
            characters (int):
            encoding (SmsCounterEncoding):
            fields (list[SmsField]):
            fill (list[int]):
            left (int):
            outside (list[Outside]):
            segments (int):
            spread (list[SmsSpread]):
            units (int):
            longest (SmsLongest | Unset):
     """

    base: int
    capacity: int
    characters: int
    encoding: SmsCounterEncoding
    fields: list[SmsField]
    fill: list[int]
    left: int
    outside: list[Outside]
    segments: int
    spread: list[SmsSpread]
    units: int
    longest: SmsLongest | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.outside import Outside # noqa: PLC0415
        from ..models.sms_field import SmsField # noqa: PLC0415
        from ..models.sms_longest import SmsLongest # noqa: PLC0415
        from ..models.sms_spread import SmsSpread # noqa: PLC0415
        base = self.base

        capacity = self.capacity

        characters = self.characters

        encoding = self.encoding.value

        fields = []
        for fields_item_data in self.fields:
            fields_item = fields_item_data.to_dict()
            fields.append(fields_item)



        fill = self.fill



        left = self.left

        outside = []
        for outside_item_data in self.outside:
            outside_item = outside_item_data.to_dict()
            outside.append(outside_item)



        segments = self.segments

        spread = []
        for spread_item_data in self.spread:
            spread_item = spread_item_data.to_dict()
            spread.append(spread_item)



        units = self.units

        longest: dict[str, Any] | Unset = UNSET
        if not isinstance(self.longest, Unset):
            longest = self.longest.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "base": base,
            "capacity": capacity,
            "characters": characters,
            "encoding": encoding,
            "fields": fields,
            "fill": fill,
            "left": left,
            "outside": outside,
            "segments": segments,
            "spread": spread,
            "units": units,
        })
        if longest is not UNSET:
            field_dict["longest"] = longest

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.outside import Outside # noqa: PLC0415
        from ..models.sms_field import SmsField # noqa: PLC0415
        from ..models.sms_longest import SmsLongest # noqa: PLC0415
        from ..models.sms_spread import SmsSpread # noqa: PLC0415
        d = dict(src_dict)
        base = d.pop("base")

        capacity = d.pop("capacity")

        characters = d.pop("characters")

        encoding = SmsCounterEncoding(d.pop("encoding"))




        fields = []
        _fields = d.pop("fields")
        for fields_item_data in (_fields):
            fields_item = SmsField.from_dict(fields_item_data)



            fields.append(fields_item)


        fill = cast(list[int], d.pop("fill"))


        left = d.pop("left")

        outside = []
        _outside = d.pop("outside")
        for outside_item_data in (_outside):
            outside_item = Outside.from_dict(outside_item_data)



            outside.append(outside_item)


        segments = d.pop("segments")

        spread = []
        _spread = d.pop("spread")
        for spread_item_data in (_spread):
            spread_item = SmsSpread.from_dict(spread_item_data)



            spread.append(spread_item)


        units = d.pop("units")

        _longest = d.pop("longest", UNSET)
        longest: SmsLongest | Unset
        if isinstance(_longest,  Unset):
            longest = UNSET
        else:
            longest = SmsLongest.from_dict(_longest)




        sms_counter = cls(
            base=base,
            capacity=capacity,
            characters=characters,
            encoding=encoding,
            fields=fields,
            fill=fill,
            left=left,
            outside=outside,
            segments=segments,
            spread=spread,
            units=units,
            longest=longest,
        )

        return sms_counter

