from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.website_hidden_block_kind import WebsiteHiddenBlockKind
from ..models.website_hidden_block_reason import WebsiteHiddenBlockReason
from ..types import UNSET, Unset






T = TypeVar("T", bound="WebsiteHiddenBlock")



@_attrs_define
class WebsiteHiddenBlock:
    """ 
        Attributes:
            kind (WebsiteHiddenBlockKind):
            line (int):
            reason (WebsiteHiddenBlockReason):
            condition (str | Unset):
            id (str | Unset):
            name (str | Unset):
     """

    kind: WebsiteHiddenBlockKind
    line: int
    reason: WebsiteHiddenBlockReason
    condition: str | Unset = UNSET
    id: str | Unset = UNSET
    name: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        kind = self.kind.value

        line = self.line

        reason = self.reason.value

        condition = self.condition

        id = self.id

        name = self.name


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "kind": kind,
            "line": line,
            "reason": reason,
        })
        if condition is not UNSET:
            field_dict["condition"] = condition
        if id is not UNSET:
            field_dict["id"] = id
        if name is not UNSET:
            field_dict["name"] = name

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        kind = WebsiteHiddenBlockKind(d.pop("kind"))




        line = d.pop("line")

        reason = WebsiteHiddenBlockReason(d.pop("reason"))




        condition = d.pop("condition", UNSET)

        id = d.pop("id", UNSET)

        name = d.pop("name", UNSET)

        website_hidden_block = cls(
            kind=kind,
            line=line,
            reason=reason,
            condition=condition,
            id=id,
            name=name,
        )

        return website_hidden_block

