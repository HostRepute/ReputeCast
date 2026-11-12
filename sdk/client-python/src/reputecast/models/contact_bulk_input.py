from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.contact_bulk_input_action import ContactBulkInputAction
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.contact_filter import ContactFilter





T = TypeVar("T", bound="ContactBulkInput")



@_attrs_define
class ContactBulkInput:
    """ 
        Attributes:
            action (ContactBulkInputAction):
            all_matching (bool | Unset):
            filter_ (ContactFilter | Unset):
            ids (list[str] | Unset):
            list_id (str | Unset): For add_to_list and remove_from_list
            reason (str | Unset): For suppress
            tag (str | Unset): For add_tag and remove_tag
     """

    action: ContactBulkInputAction
    all_matching: bool | Unset = UNSET
    filter_: ContactFilter | Unset = UNSET
    ids: list[str] | Unset = UNSET
    list_id: str | Unset = UNSET
    reason: str | Unset = UNSET
    tag: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.contact_filter import ContactFilter # noqa: PLC0415
        action = self.action.value

        all_matching = self.all_matching

        filter_: dict[str, Any] | Unset = UNSET
        if not isinstance(self.filter_, Unset):
            filter_ = self.filter_.to_dict()

        ids: list[str] | Unset = UNSET
        if not isinstance(self.ids, Unset):
            ids = self.ids



        list_id = self.list_id

        reason = self.reason

        tag = self.tag


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "action": action,
        })
        if all_matching is not UNSET:
            field_dict["allMatching"] = all_matching
        if filter_ is not UNSET:
            field_dict["filter"] = filter_
        if ids is not UNSET:
            field_dict["ids"] = ids
        if list_id is not UNSET:
            field_dict["listId"] = list_id
        if reason is not UNSET:
            field_dict["reason"] = reason
        if tag is not UNSET:
            field_dict["tag"] = tag

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.contact_filter import ContactFilter # noqa: PLC0415
        d = dict(src_dict)
        action = ContactBulkInputAction(d.pop("action"))




        all_matching = d.pop("allMatching", UNSET)

        _filter_ = d.pop("filter", UNSET)
        filter_: ContactFilter | Unset
        if isinstance(_filter_,  Unset):
            filter_ = UNSET
        else:
            filter_ = ContactFilter.from_dict(_filter_)




        ids = cast(list[str], d.pop("ids", UNSET))


        list_id = d.pop("listId", UNSET)

        reason = d.pop("reason", UNSET)

        tag = d.pop("tag", UNSET)

        contact_bulk_input = cls(
            action=action,
            all_matching=all_matching,
            filter_=filter_,
            ids=ids,
            list_id=list_id,
            reason=reason,
            tag=tag,
        )

        return contact_bulk_input

