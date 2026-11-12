from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.page_diff_line_op import PageDiffLineOp






T = TypeVar("T", bound="PageDiffLine")



@_attrs_define
class PageDiffLine:
    """ 
        Attributes:
            op (PageDiffLineOp):
            text (str):
     """

    op: PageDiffLineOp
    text: str





    def to_dict(self) -> dict[str, Any]:
        op = self.op.value

        text = self.text


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "op": op,
            "text": text,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        op = PageDiffLineOp(d.pop("op"))




        text = d.pop("text")

        page_diff_line = cls(
            op=op,
            text=text,
        )

        return page_diff_line

