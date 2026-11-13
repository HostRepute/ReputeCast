from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.countdown_ended_mode import CountdownEndedMode
from ..types import UNSET, Unset






T = TypeVar("T", bound="CountdownEnded")



@_attrs_define
class CountdownEnded:
    """ 
        Attributes:
            mode (CountdownEndedMode):
            text (str | Unset): The message shown when mode is text, up to 60 characters
     """

    mode: CountdownEndedMode
    text: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        mode = self.mode.value

        text = self.text


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "mode": mode,
        })
        if text is not UNSET:
            field_dict["text"] = text

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        mode = CountdownEndedMode(d.pop("mode"))




        text = d.pop("text", UNSET)

        countdown_ended = cls(
            mode=mode,
            text=text,
        )

        return countdown_ended

