from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.conversion_input_metadata import ConversionInputMetadata





T = TypeVar("T", bound="ConversionInput")



@_attrs_define
class ConversionInput:
    """ 
        Attributes:
            currency (str): ISO 4217: USD, EUR, GBP…
            name (str): What happened: "Upgraded to Team plan"
            value (Any): A decimal string: "29.00", not 29
            click_token (str | Unset): The rc_click token from the link the reader followed, if present
            email (str | Unset): The reader's address; send it, the click token, or both
            metadata (ConversionInputMetadata | Unset): Your own fields, kept with the conversion
            occurred_at (datetime.datetime | Unset): When it happened (now when absent): up to 30 days back
     """

    currency: str
    name: str
    value: Any
    click_token: str | Unset = UNSET
    email: str | Unset = UNSET
    metadata: ConversionInputMetadata | Unset = UNSET
    occurred_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.conversion_input_metadata import ConversionInputMetadata # noqa: PLC0415
        currency = self.currency

        name = self.name

        value = self.value

        click_token = self.click_token

        email = self.email

        metadata: dict[str, Any] | Unset = UNSET
        if not isinstance(self.metadata, Unset):
            metadata = self.metadata.to_dict()

        occurred_at: str | Unset = UNSET
        if not isinstance(self.occurred_at, Unset):
            occurred_at = self.occurred_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "currency": currency,
            "name": name,
            "value": value,
        })
        if click_token is not UNSET:
            field_dict["click_token"] = click_token
        if email is not UNSET:
            field_dict["email"] = email
        if metadata is not UNSET:
            field_dict["metadata"] = metadata
        if occurred_at is not UNSET:
            field_dict["occurred_at"] = occurred_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.conversion_input_metadata import ConversionInputMetadata # noqa: PLC0415
        d = dict(src_dict)
        currency = d.pop("currency")

        name = d.pop("name")

        value = d.pop("value")

        click_token = d.pop("click_token", UNSET)

        email = d.pop("email", UNSET)

        _metadata = d.pop("metadata", UNSET)
        metadata: ConversionInputMetadata | Unset
        if isinstance(_metadata,  Unset):
            metadata = UNSET
        else:
            metadata = ConversionInputMetadata.from_dict(_metadata)




        _occurred_at = d.pop("occurred_at", UNSET)
        occurred_at: datetime.datetime | Unset
        if isinstance(_occurred_at,  Unset):
            occurred_at = UNSET
        else:
            occurred_at = datetime.datetime.fromisoformat(_occurred_at)




        conversion_input = cls(
            currency=currency,
            name=name,
            value=value,
            click_token=click_token,
            email=email,
            metadata=metadata,
            occurred_at=occurred_at,
        )

        return conversion_input

