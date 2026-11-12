from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.provider_credentials_input_values import ProviderCredentialsInputValues





T = TypeVar("T", bound="ProviderCredentialsInput")



@_attrs_define
class ProviderCredentialsInput:
    """ 
        Attributes:
            values (ProviderCredentialsInputValues): Credential keys and their new values. Keys not listed keep their
                current value.
     """

    values: ProviderCredentialsInputValues





    def to_dict(self) -> dict[str, Any]:
        from ..models.provider_credentials_input_values import ProviderCredentialsInputValues # noqa: PLC0415
        values = self.values.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "values": values,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.provider_credentials_input_values import ProviderCredentialsInputValues # noqa: PLC0415
        d = dict(src_dict)
        values = ProviderCredentialsInputValues.from_dict(d.pop("values"))




        provider_credentials_input = cls(
            values=values,
        )

        return provider_credentials_input

