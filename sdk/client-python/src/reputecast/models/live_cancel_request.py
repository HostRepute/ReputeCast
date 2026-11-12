from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.live_cancel_request_mailbox import LiveCancelRequestMailbox
from ..types import UNSET, Unset
from uuid import UUID






T = TypeVar("T", bound="LiveCancelRequest")



@_attrs_define
class LiveCancelRequest:
    """ 
        Attributes:
            mailbox (LiveCancelRequestMailbox | Unset):
            provider_id (UUID | Unset):
     """

    mailbox: LiveCancelRequestMailbox | Unset = UNSET
    provider_id: UUID | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        mailbox: str | Unset = UNSET
        if not isinstance(self.mailbox, Unset):
            mailbox = self.mailbox.value


        provider_id: str | Unset = UNSET
        if not isinstance(self.provider_id, Unset):
            provider_id = str(self.provider_id)


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if mailbox is not UNSET:
            field_dict["mailbox"] = mailbox
        if provider_id is not UNSET:
            field_dict["providerId"] = provider_id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        _mailbox = d.pop("mailbox", UNSET)
        mailbox: LiveCancelRequestMailbox | Unset
        if isinstance(_mailbox,  Unset):
            mailbox = UNSET
        else:
            mailbox = LiveCancelRequestMailbox(_mailbox)




        _provider_id = d.pop("providerId", UNSET)
        provider_id: UUID | Unset
        if isinstance(_provider_id,  Unset):
            provider_id = UNSET
        else:
            provider_id = UUID(_provider_id)




        live_cancel_request = cls(
            mailbox=mailbox,
            provider_id=provider_id,
        )

        return live_cancel_request

