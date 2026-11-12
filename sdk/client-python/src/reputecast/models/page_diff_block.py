from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.page_diff_block_kind import PageDiffBlockKind
from typing import cast

if TYPE_CHECKING:
  from ..models.page_diff_line import PageDiffLine





T = TypeVar("T", bound="PageDiffBlock")



@_attrs_define
class PageDiffBlock:
    """ 
        Attributes:
            id (str):
            kind (PageDiffBlockKind):
            lines (list[PageDiffLine]):
            name (str):
     """

    id: str
    kind: PageDiffBlockKind
    lines: list[PageDiffLine]
    name: str





    def to_dict(self) -> dict[str, Any]:
        from ..models.page_diff_line import PageDiffLine # noqa: PLC0415
        id = self.id

        kind = self.kind.value

        lines = []
        for lines_item_data in self.lines:
            lines_item = lines_item_data.to_dict()
            lines.append(lines_item)



        name = self.name


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "kind": kind,
            "lines": lines,
            "name": name,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.page_diff_line import PageDiffLine # noqa: PLC0415
        d = dict(src_dict)
        id = d.pop("id")

        kind = PageDiffBlockKind(d.pop("kind"))




        lines = []
        _lines = d.pop("lines")
        for lines_item_data in (_lines):
            lines_item = PageDiffLine.from_dict(lines_item_data)



            lines.append(lines_item)


        name = d.pop("name")

        page_diff_block = cls(
            id=id,
            kind=kind,
            lines=lines,
            name=name,
        )

        return page_diff_block

