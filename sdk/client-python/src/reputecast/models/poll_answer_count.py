from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="PollAnswerCount")



@_attrs_define
class PollAnswerCount:
    """ 
        Attributes:
            count (int):
            key (int):
            label (str):
            percent (float):
            emoji (str | Unset):
            tag (str | Unset):
     """

    count: int
    key: int
    label: str
    percent: float
    emoji: str | Unset = UNSET
    tag: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        count = self.count

        key = self.key

        label = self.label

        percent = self.percent

        emoji = self.emoji

        tag = self.tag


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "count": count,
            "key": key,
            "label": label,
            "percent": percent,
        })
        if emoji is not UNSET:
            field_dict["emoji"] = emoji
        if tag is not UNSET:
            field_dict["tag"] = tag

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        count = d.pop("count")

        key = d.pop("key")

        label = d.pop("label")

        percent = d.pop("percent")

        emoji = d.pop("emoji", UNSET)

        tag = d.pop("tag", UNSET)

        poll_answer_count = cls(
            count=count,
            key=key,
            label=label,
            percent=percent,
            emoji=emoji,
            tag=tag,
        )

        return poll_answer_count

