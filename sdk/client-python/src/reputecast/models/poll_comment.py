from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.poll_comment_group import PollCommentGroup
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="PollComment")



@_attrs_define
class PollComment:
    """ 
        Attributes:
            answer (int):
            at (datetime.datetime):
            comment (str):
            id (str):
            label (str):
            non_contact (bool):
            contact (str | Unset):
            contact_id (str | Unset):
            group (PollCommentGroup | Unset):
     """

    answer: int
    at: datetime.datetime
    comment: str
    id: str
    label: str
    non_contact: bool
    contact: str | Unset = UNSET
    contact_id: str | Unset = UNSET
    group: PollCommentGroup | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        answer = self.answer

        at = self.at.isoformat()

        comment = self.comment

        id = self.id

        label = self.label

        non_contact = self.non_contact

        contact = self.contact

        contact_id = self.contact_id

        group: str | Unset = UNSET
        if not isinstance(self.group, Unset):
            group = self.group.value



        field_dict: dict[str, Any] = {}

        field_dict.update({
            "answer": answer,
            "at": at,
            "comment": comment,
            "id": id,
            "label": label,
            "nonContact": non_contact,
        })
        if contact is not UNSET:
            field_dict["contact"] = contact
        if contact_id is not UNSET:
            field_dict["contactId"] = contact_id
        if group is not UNSET:
            field_dict["group"] = group

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        answer = d.pop("answer")

        at = datetime.datetime.fromisoformat(d.pop("at"))




        comment = d.pop("comment")

        id = d.pop("id")

        label = d.pop("label")

        non_contact = d.pop("nonContact")

        contact = d.pop("contact", UNSET)

        contact_id = d.pop("contactId", UNSET)

        _group = d.pop("group", UNSET)
        group: PollCommentGroup | Unset
        if isinstance(_group,  Unset):
            group = UNSET
        else:
            group = PollCommentGroup(_group)




        poll_comment = cls(
            answer=answer,
            at=at,
            comment=comment,
            id=id,
            label=label,
            non_contact=non_contact,
            contact=contact,
            contact_id=contact_id,
            group=group,
        )

        return poll_comment

