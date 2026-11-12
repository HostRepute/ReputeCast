from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.sending_domain_detail_status import SendingDomainDetailStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.provider_setup_note import ProviderSetupNote
  from ..models.sender_provider_column import SenderProviderColumn
  from ..models.sending_domain_counts import SendingDomainCounts
  from ..models.sending_domain_record import SendingDomainRecord
  from ..models.sending_domain_tracking import SendingDomainTracking





T = TypeVar("T", bound="SendingDomainDetail")



@_attrs_define
class SendingDomainDetail:
    """ 
        Attributes:
            bimi (str):
            can_manage (bool):
            cloudflare (bool):
            counts (SendingDomainCounts):
            dmarc_tip (str):
            dns_host (str):
            domain (str):
            helpers (list[str]):
            id (str):
            nameservers (list[str]):
            note (str):
            notes (list[ProviderSetupNote]):
            providers (list[SenderProviderColumn]):
            records (list[SendingDomainRecord]):
            senders (int):
            status (SendingDomainDetailStatus):
            tracking (SendingDomainTracking):
            checked_at (datetime.datetime | Unset):
     """

    bimi: str
    can_manage: bool
    cloudflare: bool
    counts: SendingDomainCounts
    dmarc_tip: str
    dns_host: str
    domain: str
    helpers: list[str]
    id: str
    nameservers: list[str]
    note: str
    notes: list[ProviderSetupNote]
    providers: list[SenderProviderColumn]
    records: list[SendingDomainRecord]
    senders: int
    status: SendingDomainDetailStatus
    tracking: SendingDomainTracking
    checked_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.provider_setup_note import ProviderSetupNote # noqa: PLC0415
        from ..models.sender_provider_column import SenderProviderColumn # noqa: PLC0415
        from ..models.sending_domain_counts import SendingDomainCounts # noqa: PLC0415
        from ..models.sending_domain_record import SendingDomainRecord # noqa: PLC0415
        from ..models.sending_domain_tracking import SendingDomainTracking # noqa: PLC0415
        bimi = self.bimi

        can_manage = self.can_manage

        cloudflare = self.cloudflare

        counts = self.counts.to_dict()

        dmarc_tip = self.dmarc_tip

        dns_host = self.dns_host

        domain = self.domain

        helpers = self.helpers



        id = self.id

        nameservers = self.nameservers



        note = self.note

        notes = []
        for notes_item_data in self.notes:
            notes_item = notes_item_data.to_dict()
            notes.append(notes_item)



        providers = []
        for providers_item_data in self.providers:
            providers_item = providers_item_data.to_dict()
            providers.append(providers_item)



        records = []
        for records_item_data in self.records:
            records_item = records_item_data.to_dict()
            records.append(records_item)



        senders = self.senders

        status = self.status.value

        tracking = self.tracking.to_dict()

        checked_at: str | Unset = UNSET
        if not isinstance(self.checked_at, Unset):
            checked_at = self.checked_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "bimi": bimi,
            "canManage": can_manage,
            "cloudflare": cloudflare,
            "counts": counts,
            "dmarcTip": dmarc_tip,
            "dnsHost": dns_host,
            "domain": domain,
            "helpers": helpers,
            "id": id,
            "nameservers": nameservers,
            "note": note,
            "notes": notes,
            "providers": providers,
            "records": records,
            "senders": senders,
            "status": status,
            "tracking": tracking,
        })
        if checked_at is not UNSET:
            field_dict["checkedAt"] = checked_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.provider_setup_note import ProviderSetupNote # noqa: PLC0415
        from ..models.sender_provider_column import SenderProviderColumn # noqa: PLC0415
        from ..models.sending_domain_counts import SendingDomainCounts # noqa: PLC0415
        from ..models.sending_domain_record import SendingDomainRecord # noqa: PLC0415
        from ..models.sending_domain_tracking import SendingDomainTracking # noqa: PLC0415
        d = dict(src_dict)
        bimi = d.pop("bimi")

        can_manage = d.pop("canManage")

        cloudflare = d.pop("cloudflare")

        counts = SendingDomainCounts.from_dict(d.pop("counts"))




        dmarc_tip = d.pop("dmarcTip")

        dns_host = d.pop("dnsHost")

        domain = d.pop("domain")

        helpers = cast(list[str], d.pop("helpers"))


        id = d.pop("id")

        nameservers = cast(list[str], d.pop("nameservers"))


        note = d.pop("note")

        notes = []
        _notes = d.pop("notes")
        for notes_item_data in (_notes):
            notes_item = ProviderSetupNote.from_dict(notes_item_data)



            notes.append(notes_item)


        providers = []
        _providers = d.pop("providers")
        for providers_item_data in (_providers):
            providers_item = SenderProviderColumn.from_dict(providers_item_data)



            providers.append(providers_item)


        records = []
        _records = d.pop("records")
        for records_item_data in (_records):
            records_item = SendingDomainRecord.from_dict(records_item_data)



            records.append(records_item)


        senders = d.pop("senders")

        status = SendingDomainDetailStatus(d.pop("status"))




        tracking = SendingDomainTracking.from_dict(d.pop("tracking"))




        _checked_at = d.pop("checkedAt", UNSET)
        checked_at: datetime.datetime | Unset
        if isinstance(_checked_at,  Unset):
            checked_at = UNSET
        else:
            checked_at = datetime.datetime.fromisoformat(_checked_at)




        sending_domain_detail = cls(
            bimi=bimi,
            can_manage=can_manage,
            cloudflare=cloudflare,
            counts=counts,
            dmarc_tip=dmarc_tip,
            dns_host=dns_host,
            domain=domain,
            helpers=helpers,
            id=id,
            nameservers=nameservers,
            note=note,
            notes=notes,
            providers=providers,
            records=records,
            senders=senders,
            status=status,
            tracking=tracking,
            checked_at=checked_at,
        )

        return sending_domain_detail

