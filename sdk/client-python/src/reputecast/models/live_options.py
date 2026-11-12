from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="LiveOptions")



@_attrs_define
class LiveOptions:
    """ 
        Attributes:
            can_cancel (bool):
            can_pause (bool):
            can_resume (bool):
     """

    can_cancel: bool
    can_pause: bool
    can_resume: bool





    def to_dict(self) -> dict[str, Any]:
        can_cancel = self.can_cancel

        can_pause = self.can_pause

        can_resume = self.can_resume


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canCancel": can_cancel,
            "canPause": can_pause,
            "canResume": can_resume,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        can_cancel = d.pop("canCancel")

        can_pause = d.pop("canPause")

        can_resume = d.pop("canResume")

        live_options = cls(
            can_cancel=can_cancel,
            can_pause=can_pause,
            can_resume=can_resume,
        )

        return live_options

