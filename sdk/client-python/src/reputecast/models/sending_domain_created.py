from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.provider_setup_note import ProviderSetupNote
  from ..models.sending_domain_detail import SendingDomainDetail





T = TypeVar("T", bound="SendingDomainCreated")



@_attrs_define
class SendingDomainCreated:
    """ 
        Attributes:
            domain (SendingDomainDetail):
            notes (list[ProviderSetupNote]):
     """

    domain: SendingDomainDetail
    notes: list[ProviderSetupNote]





    def to_dict(self) -> dict[str, Any]:
        from ..models.provider_setup_note import ProviderSetupNote # noqa: PLC0415
        from ..models.sending_domain_detail import SendingDomainDetail # noqa: PLC0415
        domain = self.domain.to_dict()

        notes = []
        for notes_item_data in self.notes:
            notes_item = notes_item_data.to_dict()
            notes.append(notes_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "domain": domain,
            "notes": notes,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.provider_setup_note import ProviderSetupNote # noqa: PLC0415
        from ..models.sending_domain_detail import SendingDomainDetail # noqa: PLC0415
        d = dict(src_dict)
        domain = SendingDomainDetail.from_dict(d.pop("domain"))




        notes = []
        _notes = d.pop("notes")
        for notes_item_data in (_notes):
            notes_item = ProviderSetupNote.from_dict(notes_item_data)



            notes.append(notes_item)


        sending_domain_created = cls(
            domain=domain,
            notes=notes,
        )

        return sending_domain_created

