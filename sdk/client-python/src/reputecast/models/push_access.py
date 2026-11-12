from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="PushAccess")



@_attrs_define
class PushAccess:
    """ 
        Attributes:
            can_edit (bool):
            can_send (bool):
            helpers (list[str]):
            missing (list[str]):
     """

    can_edit: bool
    can_send: bool
    helpers: list[str]
    missing: list[str]





    def to_dict(self) -> dict[str, Any]:
        can_edit = self.can_edit

        can_send = self.can_send

        helpers = self.helpers



        missing = self.missing




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canEdit": can_edit,
            "canSend": can_send,
            "helpers": helpers,
            "missing": missing,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        can_edit = d.pop("canEdit")

        can_send = d.pop("canSend")

        helpers = cast(list[str], d.pop("helpers"))


        missing = cast(list[str], d.pop("missing"))


        push_access = cls(
            can_edit=can_edit,
            can_send=can_send,
            helpers=helpers,
            missing=missing,
        )

        return push_access

