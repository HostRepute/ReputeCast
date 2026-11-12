from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.composer_comment import ComposerComment





T = TypeVar("T", bound="ComposerThread")



@_attrs_define
class ComposerThread:
    """ 
        Attributes:
            anchor (str):
            anchor_label (str):
            comments (list[ComposerComment]):
            id (str):
            resolved_at (datetime.datetime | Unset):
            resolved_by (str | Unset):
     """

    anchor: str
    anchor_label: str
    comments: list[ComposerComment]
    id: str
    resolved_at: datetime.datetime | Unset = UNSET
    resolved_by: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.composer_comment import ComposerComment # noqa: PLC0415
        anchor = self.anchor

        anchor_label = self.anchor_label

        comments = []
        for comments_item_data in self.comments:
            comments_item = comments_item_data.to_dict()
            comments.append(comments_item)



        id = self.id

        resolved_at: str | Unset = UNSET
        if not isinstance(self.resolved_at, Unset):
            resolved_at = self.resolved_at.isoformat()

        resolved_by = self.resolved_by


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "anchor": anchor,
            "anchorLabel": anchor_label,
            "comments": comments,
            "id": id,
        })
        if resolved_at is not UNSET:
            field_dict["resolvedAt"] = resolved_at
        if resolved_by is not UNSET:
            field_dict["resolvedBy"] = resolved_by

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.composer_comment import ComposerComment # noqa: PLC0415
        d = dict(src_dict)
        anchor = d.pop("anchor")

        anchor_label = d.pop("anchorLabel")

        comments = []
        _comments = d.pop("comments")
        for comments_item_data in (_comments):
            comments_item = ComposerComment.from_dict(comments_item_data)



            comments.append(comments_item)


        id = d.pop("id")

        _resolved_at = d.pop("resolvedAt", UNSET)
        resolved_at: datetime.datetime | Unset
        if isinstance(_resolved_at,  Unset):
            resolved_at = UNSET
        else:
            resolved_at = datetime.datetime.fromisoformat(_resolved_at)




        resolved_by = d.pop("resolvedBy", UNSET)

        composer_thread = cls(
            anchor=anchor,
            anchor_label=anchor_label,
            comments=comments,
            id=id,
            resolved_at=resolved_at,
            resolved_by=resolved_by,
        )

        return composer_thread

