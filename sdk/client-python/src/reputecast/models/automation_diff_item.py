from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.automation_diff_item_kind import AutomationDiffItemKind
from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="AutomationDiffItem")



@_attrs_define
class AutomationDiffItem:
    """ 
        Attributes:
            kind (AutomationDiffItemKind):
            node_ids (list[str]):
            title (str):
            detail (str | Unset):
     """

    kind: AutomationDiffItemKind
    node_ids: list[str]
    title: str
    detail: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        kind = self.kind.value

        node_ids = self.node_ids



        title = self.title

        detail = self.detail


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "kind": kind,
            "nodeIds": node_ids,
            "title": title,
        })
        if detail is not UNSET:
            field_dict["detail"] = detail

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        kind = AutomationDiffItemKind(d.pop("kind"))




        node_ids = cast(list[str], d.pop("nodeIds"))


        title = d.pop("title")

        detail = d.pop("detail", UNSET)

        automation_diff_item = cls(
            kind=kind,
            node_ids=node_ids,
            title=title,
            detail=detail,
        )

        return automation_diff_item

