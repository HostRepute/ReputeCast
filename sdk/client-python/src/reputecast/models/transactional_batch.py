from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.transactional_send import TransactionalSend





T = TypeVar("T", bound="TransactionalBatch")



@_attrs_define
class TransactionalBatch:
    """ 
        Attributes:
            messages (list[TransactionalSend]):
     """

    messages: list[TransactionalSend]





    def to_dict(self) -> dict[str, Any]:
        from ..models.transactional_send import TransactionalSend # noqa: PLC0415
        messages = []
        for messages_item_data in self.messages:
            messages_item = messages_item_data.to_dict()
            messages.append(messages_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "messages": messages,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.transactional_send import TransactionalSend # noqa: PLC0415
        d = dict(src_dict)
        messages = []
        _messages = d.pop("messages")
        for messages_item_data in (_messages):
            messages_item = TransactionalSend.from_dict(messages_item_data)



            messages.append(messages_item)


        transactional_batch = cls(
            messages=messages,
        )

        return transactional_batch

