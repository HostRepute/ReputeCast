from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="AddWorkspaceLanguage")



@_attrs_define
class AddWorkspaceLanguage:
    """ 
        Attributes:
            code (str): A base language code, such as fr
     """

    code: str





    def to_dict(self) -> dict[str, Any]:
        code = self.code


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "code": code,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        code = d.pop("code")

        add_workspace_language = cls(
            code=code,
        )

        return add_workspace_language

