from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.ai_personal_account_provider import AIPersonalAccountProvider
from ..models.ai_personal_account_status import AIPersonalAccountStatus
from typing import cast






T = TypeVar("T", bound="AIPersonalAccount")



@_attrs_define
class AIPersonalAccount:
    """ 
        Attributes:
            id (str):
            models (list[str]):
            provider (AIPersonalAccountProvider):
            provider_name (str):
            status (AIPersonalAccountStatus):
     """

    id: str
    models: list[str]
    provider: AIPersonalAccountProvider
    provider_name: str
    status: AIPersonalAccountStatus





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        models = self.models



        provider = self.provider.value

        provider_name = self.provider_name

        status = self.status.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "models": models,
            "provider": provider,
            "providerName": provider_name,
            "status": status,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        models = cast(list[str], d.pop("models"))


        provider = AIPersonalAccountProvider(d.pop("provider"))




        provider_name = d.pop("providerName")

        status = AIPersonalAccountStatus(d.pop("status"))




        ai_personal_account = cls(
            id=id,
            models=models,
            provider=provider,
            provider_name=provider_name,
            status=status,
        )

        return ai_personal_account

