from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.block_segment import BlockSegment





T = TypeVar("T", bound="CompareBlock")



@_attrs_define
class CompareBlock:
    """ 
        Attributes:
            index (int):
            name (str):
            segments (list[BlockSegment]):
            text (str):
     """

    index: int
    name: str
    segments: list[BlockSegment]
    text: str





    def to_dict(self) -> dict[str, Any]:
        from ..models.block_segment import BlockSegment # noqa: PLC0415
        index = self.index

        name = self.name

        segments = []
        for segments_item_data in self.segments:
            segments_item = segments_item_data.to_dict()
            segments.append(segments_item)



        text = self.text


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "index": index,
            "name": name,
            "segments": segments,
            "text": text,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.block_segment import BlockSegment # noqa: PLC0415
        d = dict(src_dict)
        index = d.pop("index")

        name = d.pop("name")

        segments = []
        _segments = d.pop("segments")
        for segments_item_data in (_segments):
            segments_item = BlockSegment.from_dict(segments_item_data)



            segments.append(segments_item)


        text = d.pop("text")

        compare_block = cls(
            index=index,
            name=name,
            segments=segments,
            text=text,
        )

        return compare_block

