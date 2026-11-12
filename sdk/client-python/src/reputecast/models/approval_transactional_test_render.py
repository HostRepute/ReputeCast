from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ApprovalTransactionalTestRender")



@_attrs_define
class ApprovalTransactionalTestRender:
    """ 
        Attributes:
            name (str):
            ok (bool):
     """

    name: str
    ok: bool





    def to_dict(self) -> dict[str, Any]:
        name = self.name

        ok = self.ok


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "name": name,
            "ok": ok,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        name = d.pop("name")

        ok = d.pop("ok")

        approval_transactional_test_render = cls(
            name=name,
            ok=ok,
        )

        return approval_transactional_test_render

