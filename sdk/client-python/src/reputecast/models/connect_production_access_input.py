from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.connect_production_access_input_language import ConnectProductionAccessInputLanguage
from ..models.connect_production_access_input_mail_type import ConnectProductionAccessInputMailType






T = TypeVar("T", bound="ConnectProductionAccessInput")



@_attrs_define
class ConnectProductionAccessInput:
    """ 
        Attributes:
            daily_volume (int):
            language (ConnectProductionAccessInputLanguage):
            mail_type (ConnectProductionAccessInputMailType):
            region (str):
            use_case (str):
            website (str):
     """

    daily_volume: int
    language: ConnectProductionAccessInputLanguage
    mail_type: ConnectProductionAccessInputMailType
    region: str
    use_case: str
    website: str





    def to_dict(self) -> dict[str, Any]:
        daily_volume = self.daily_volume

        language = self.language.value

        mail_type = self.mail_type.value

        region = self.region

        use_case = self.use_case

        website = self.website


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "dailyVolume": daily_volume,
            "language": language,
            "mailType": mail_type,
            "region": region,
            "useCase": use_case,
            "website": website,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        daily_volume = d.pop("dailyVolume")

        language = ConnectProductionAccessInputLanguage(d.pop("language"))




        mail_type = ConnectProductionAccessInputMailType(d.pop("mailType"))




        region = d.pop("region")

        use_case = d.pop("useCase")

        website = d.pop("website")

        connect_production_access_input = cls(
            daily_volume=daily_volume,
            language=language,
            mail_type=mail_type,
            region=region,
            use_case=use_case,
            website=website,
        )

        return connect_production_access_input

