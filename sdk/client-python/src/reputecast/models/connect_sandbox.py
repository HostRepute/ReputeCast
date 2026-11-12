from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.connect_sandbox_language import ConnectSandboxLanguage
from ..models.connect_sandbox_mail_type import ConnectSandboxMailType
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ConnectSandbox")



@_attrs_define
class ConnectSandbox:
    """ 
        Attributes:
            daily_volume (int):
            language (ConnectSandboxLanguage):
            mail_type (ConnectSandboxMailType):
            name (str):
            region (str):
            selected (bool):
            use_case (str):
            website (str):
            requested_at (datetime.datetime | Unset):
     """

    daily_volume: int
    language: ConnectSandboxLanguage
    mail_type: ConnectSandboxMailType
    name: str
    region: str
    selected: bool
    use_case: str
    website: str
    requested_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        daily_volume = self.daily_volume

        language = self.language.value

        mail_type = self.mail_type.value

        name = self.name

        region = self.region

        selected = self.selected

        use_case = self.use_case

        website = self.website

        requested_at: str | Unset = UNSET
        if not isinstance(self.requested_at, Unset):
            requested_at = self.requested_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "dailyVolume": daily_volume,
            "language": language,
            "mailType": mail_type,
            "name": name,
            "region": region,
            "selected": selected,
            "useCase": use_case,
            "website": website,
        })
        if requested_at is not UNSET:
            field_dict["requestedAt"] = requested_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        daily_volume = d.pop("dailyVolume")

        language = ConnectSandboxLanguage(d.pop("language"))




        mail_type = ConnectSandboxMailType(d.pop("mailType"))




        name = d.pop("name")

        region = d.pop("region")

        selected = d.pop("selected")

        use_case = d.pop("useCase")

        website = d.pop("website")

        _requested_at = d.pop("requestedAt", UNSET)
        requested_at: datetime.datetime | Unset
        if isinstance(_requested_at,  Unset):
            requested_at = UNSET
        else:
            requested_at = datetime.datetime.fromisoformat(_requested_at)




        connect_sandbox = cls(
            daily_volume=daily_volume,
            language=language,
            mail_type=mail_type,
            name=name,
            region=region,
            selected=selected,
            use_case=use_case,
            website=website,
            requested_at=requested_at,
        )

        return connect_sandbox

