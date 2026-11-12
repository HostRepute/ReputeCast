from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.live_resume_request_mailbox import LiveResumeRequestMailbox
from ..types import UNSET, Unset
from uuid import UUID






T = TypeVar("T", bound="LiveResumeRequest")



@_attrs_define
class LiveResumeRequest:
    """ 
        Attributes:
            limit (float | Unset):
            mailbox (LiveResumeRequestMailbox | Unset):
            provider_id (UUID | Unset):
            rate (float | Unset):
            with_changes (bool | Unset):
     """

    limit: float | Unset = UNSET
    mailbox: LiveResumeRequestMailbox | Unset = UNSET
    provider_id: UUID | Unset = UNSET
    rate: float | Unset = UNSET
    with_changes: bool | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        limit = self.limit

        mailbox: str | Unset = UNSET
        if not isinstance(self.mailbox, Unset):
            mailbox = self.mailbox.value


        provider_id: str | Unset = UNSET
        if not isinstance(self.provider_id, Unset):
            provider_id = str(self.provider_id)

        rate = self.rate

        with_changes = self.with_changes


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if limit is not UNSET:
            field_dict["limit"] = limit
        if mailbox is not UNSET:
            field_dict["mailbox"] = mailbox
        if provider_id is not UNSET:
            field_dict["providerId"] = provider_id
        if rate is not UNSET:
            field_dict["rate"] = rate
        if with_changes is not UNSET:
            field_dict["withChanges"] = with_changes

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        limit = d.pop("limit", UNSET)

        _mailbox = d.pop("mailbox", UNSET)
        mailbox: LiveResumeRequestMailbox | Unset
        if isinstance(_mailbox,  Unset):
            mailbox = UNSET
        else:
            mailbox = LiveResumeRequestMailbox(_mailbox)




        _provider_id = d.pop("providerId", UNSET)
        provider_id: UUID | Unset
        if isinstance(_provider_id,  Unset):
            provider_id = UNSET
        else:
            provider_id = UUID(_provider_id)




        rate = d.pop("rate", UNSET)

        with_changes = d.pop("withChanges", UNSET)

        live_resume_request = cls(
            limit=limit,
            mailbox=mailbox,
            provider_id=provider_id,
            rate=rate,
            with_changes=with_changes,
        )

        return live_resume_request

