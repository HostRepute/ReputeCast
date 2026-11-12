from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="LiveHelpers")



@_attrs_define
class LiveHelpers:
    """ 
        Attributes:
            approve (list[str]):
            pause (list[str]):
            send (list[str]):
     """

    approve: list[str]
    pause: list[str]
    send: list[str]





    def to_dict(self) -> dict[str, Any]:
        approve = self.approve



        pause = self.pause



        send = self.send




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "approve": approve,
            "pause": pause,
            "send": send,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        approve = cast(list[str], d.pop("approve"))


        pause = cast(list[str], d.pop("pause"))


        send = cast(list[str], d.pop("send"))


        live_helpers = cls(
            approve=approve,
            pause=pause,
            send=send,
        )

        return live_helpers

