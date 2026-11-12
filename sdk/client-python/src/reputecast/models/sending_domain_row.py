from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.sending_domain_row_status import SendingDomainRowStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.sender_domain_cell import SenderDomainCell





T = TypeVar("T", bound="SendingDomainRow")



@_attrs_define
class SendingDomainRow:
    """ 
        Attributes:
            bimi (SenderDomainCell):
            dkim (SenderDomainCell):
            dmarc (SenderDomainCell):
            domain (str):
            id (str):
            note (str):
            return_path (SenderDomainCell):
            senders (int):
            spf (SenderDomainCell):
            status (SendingDomainRowStatus):
            tracking (SenderDomainCell):
            transactional (bool):
            used_by (list[str]):
            checked_at (datetime.datetime | Unset):
     """

    bimi: SenderDomainCell
    dkim: SenderDomainCell
    dmarc: SenderDomainCell
    domain: str
    id: str
    note: str
    return_path: SenderDomainCell
    senders: int
    spf: SenderDomainCell
    status: SendingDomainRowStatus
    tracking: SenderDomainCell
    transactional: bool
    used_by: list[str]
    checked_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.sender_domain_cell import SenderDomainCell # noqa: PLC0415
        bimi = self.bimi.to_dict()

        dkim = self.dkim.to_dict()

        dmarc = self.dmarc.to_dict()

        domain = self.domain

        id = self.id

        note = self.note

        return_path = self.return_path.to_dict()

        senders = self.senders

        spf = self.spf.to_dict()

        status = self.status.value

        tracking = self.tracking.to_dict()

        transactional = self.transactional

        used_by = self.used_by



        checked_at: str | Unset = UNSET
        if not isinstance(self.checked_at, Unset):
            checked_at = self.checked_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "bimi": bimi,
            "dkim": dkim,
            "dmarc": dmarc,
            "domain": domain,
            "id": id,
            "note": note,
            "returnPath": return_path,
            "senders": senders,
            "spf": spf,
            "status": status,
            "tracking": tracking,
            "transactional": transactional,
            "usedBy": used_by,
        })
        if checked_at is not UNSET:
            field_dict["checkedAt"] = checked_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.sender_domain_cell import SenderDomainCell # noqa: PLC0415
        d = dict(src_dict)
        bimi = SenderDomainCell.from_dict(d.pop("bimi"))




        dkim = SenderDomainCell.from_dict(d.pop("dkim"))




        dmarc = SenderDomainCell.from_dict(d.pop("dmarc"))




        domain = d.pop("domain")

        id = d.pop("id")

        note = d.pop("note")

        return_path = SenderDomainCell.from_dict(d.pop("returnPath"))




        senders = d.pop("senders")

        spf = SenderDomainCell.from_dict(d.pop("spf"))




        status = SendingDomainRowStatus(d.pop("status"))




        tracking = SenderDomainCell.from_dict(d.pop("tracking"))




        transactional = d.pop("transactional")

        used_by = cast(list[str], d.pop("usedBy"))


        _checked_at = d.pop("checkedAt", UNSET)
        checked_at: datetime.datetime | Unset
        if isinstance(_checked_at,  Unset):
            checked_at = UNSET
        else:
            checked_at = datetime.datetime.fromisoformat(_checked_at)




        sending_domain_row = cls(
            bimi=bimi,
            dkim=dkim,
            dmarc=dmarc,
            domain=domain,
            id=id,
            note=note,
            return_path=return_path,
            senders=senders,
            spf=spf,
            status=status,
            tracking=tracking,
            transactional=transactional,
            used_by=used_by,
            checked_at=checked_at,
        )

        return sending_domain_row

