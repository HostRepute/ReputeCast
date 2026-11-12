from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.sms_registration_view_kind import SmsRegistrationViewKind
from ..models.sms_registration_view_source import SmsRegistrationViewSource
from ..models.sms_registration_view_status import SmsRegistrationViewStatus
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="SmsRegistrationView")



@_attrs_define
class SmsRegistrationView:
    """ 
        Attributes:
            country (str):
            kind (SmsRegistrationViewKind):
            source (SmsRegistrationViewSource):
            status (SmsRegistrationViewStatus):
            name (str | Unset):
            read_at (datetime.datetime | Unset):
            reason (str | Unset):
            set_by (str | Unset):
            since (datetime.datetime | Unset):
     """

    country: str
    kind: SmsRegistrationViewKind
    source: SmsRegistrationViewSource
    status: SmsRegistrationViewStatus
    name: str | Unset = UNSET
    read_at: datetime.datetime | Unset = UNSET
    reason: str | Unset = UNSET
    set_by: str | Unset = UNSET
    since: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        country = self.country

        kind = self.kind.value

        source = self.source.value

        status = self.status.value

        name = self.name

        read_at: str | Unset = UNSET
        if not isinstance(self.read_at, Unset):
            read_at = self.read_at.isoformat()

        reason = self.reason

        set_by = self.set_by

        since: str | Unset = UNSET
        if not isinstance(self.since, Unset):
            since = self.since.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "country": country,
            "kind": kind,
            "source": source,
            "status": status,
        })
        if name is not UNSET:
            field_dict["name"] = name
        if read_at is not UNSET:
            field_dict["readAt"] = read_at
        if reason is not UNSET:
            field_dict["reason"] = reason
        if set_by is not UNSET:
            field_dict["setBy"] = set_by
        if since is not UNSET:
            field_dict["since"] = since

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        country = d.pop("country")

        kind = SmsRegistrationViewKind(d.pop("kind"))




        source = SmsRegistrationViewSource(d.pop("source"))




        status = SmsRegistrationViewStatus(d.pop("status"))




        name = d.pop("name", UNSET)

        _read_at = d.pop("readAt", UNSET)
        read_at: datetime.datetime | Unset
        if isinstance(_read_at,  Unset):
            read_at = UNSET
        else:
            read_at = datetime.datetime.fromisoformat(_read_at)




        reason = d.pop("reason", UNSET)

        set_by = d.pop("setBy", UNSET)

        _since = d.pop("since", UNSET)
        since: datetime.datetime | Unset
        if isinstance(_since,  Unset):
            since = UNSET
        else:
            since = datetime.datetime.fromisoformat(_since)




        sms_registration_view = cls(
            country=country,
            kind=kind,
            source=source,
            status=status,
            name=name,
            read_at=read_at,
            reason=reason,
            set_by=set_by,
            since=since,
        )

        return sms_registration_view

