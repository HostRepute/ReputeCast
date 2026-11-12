from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.transactional_contact_category import TransactionalContactCategory
  from ..models.transactional_contact_message import TransactionalContactMessage
  from ..models.transactional_contact_opt_out import TransactionalContactOptOut





T = TypeVar("T", bound="TransactionalContactActivity")



@_attrs_define
class TransactionalContactActivity:
    """ 
        Attributes:
            can_view_content (bool):
            categories (list[TransactionalContactCategory]):
            messages (list[TransactionalContactMessage]):
            opt_outs (list[TransactionalContactOptOut]):
     """

    can_view_content: bool
    categories: list[TransactionalContactCategory]
    messages: list[TransactionalContactMessage]
    opt_outs: list[TransactionalContactOptOut]





    def to_dict(self) -> dict[str, Any]:
        from ..models.transactional_contact_category import TransactionalContactCategory # noqa: PLC0415
        from ..models.transactional_contact_message import TransactionalContactMessage # noqa: PLC0415
        from ..models.transactional_contact_opt_out import TransactionalContactOptOut # noqa: PLC0415
        can_view_content = self.can_view_content

        categories = []
        for categories_item_data in self.categories:
            categories_item = categories_item_data.to_dict()
            categories.append(categories_item)



        messages = []
        for messages_item_data in self.messages:
            messages_item = messages_item_data.to_dict()
            messages.append(messages_item)



        opt_outs = []
        for opt_outs_item_data in self.opt_outs:
            opt_outs_item = opt_outs_item_data.to_dict()
            opt_outs.append(opt_outs_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canViewContent": can_view_content,
            "categories": categories,
            "messages": messages,
            "optOuts": opt_outs,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.transactional_contact_category import TransactionalContactCategory # noqa: PLC0415
        from ..models.transactional_contact_message import TransactionalContactMessage # noqa: PLC0415
        from ..models.transactional_contact_opt_out import TransactionalContactOptOut # noqa: PLC0415
        d = dict(src_dict)
        can_view_content = d.pop("canViewContent")

        categories = []
        _categories = d.pop("categories")
        for categories_item_data in (_categories):
            categories_item = TransactionalContactCategory.from_dict(categories_item_data)



            categories.append(categories_item)


        messages = []
        _messages = d.pop("messages")
        for messages_item_data in (_messages):
            messages_item = TransactionalContactMessage.from_dict(messages_item_data)



            messages.append(messages_item)


        opt_outs = []
        _opt_outs = d.pop("optOuts")
        for opt_outs_item_data in (_opt_outs):
            opt_outs_item = TransactionalContactOptOut.from_dict(opt_outs_item_data)



            opt_outs.append(opt_outs_item)


        transactional_contact_activity = cls(
            can_view_content=can_view_content,
            categories=categories,
            messages=messages,
            opt_outs=opt_outs,
        )

        return transactional_contact_activity

