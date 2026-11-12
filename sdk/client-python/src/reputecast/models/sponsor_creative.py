from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.sponsor_ut_ms import SponsorUTMs





T = TypeVar("T", bound="SponsorCreative")



@_attrs_define
class SponsorCreative:
    """ 
        Attributes:
            utm (SponsorUTMs):
            button (str | Unset):
            headline (str | Unset):
            image (str | Unset):
            link (str | Unset):
            logo (str | Unset):
            text (str | Unset): Up to 280 characters.
     """

    utm: SponsorUTMs
    button: str | Unset = UNSET
    headline: str | Unset = UNSET
    image: str | Unset = UNSET
    link: str | Unset = UNSET
    logo: str | Unset = UNSET
    text: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.sponsor_ut_ms import SponsorUTMs # noqa: PLC0415
        utm = self.utm.to_dict()

        button = self.button

        headline = self.headline

        image = self.image

        link = self.link

        logo = self.logo

        text = self.text


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "utm": utm,
        })
        if button is not UNSET:
            field_dict["button"] = button
        if headline is not UNSET:
            field_dict["headline"] = headline
        if image is not UNSET:
            field_dict["image"] = image
        if link is not UNSET:
            field_dict["link"] = link
        if logo is not UNSET:
            field_dict["logo"] = logo
        if text is not UNSET:
            field_dict["text"] = text

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.sponsor_ut_ms import SponsorUTMs # noqa: PLC0415
        d = dict(src_dict)
        utm = SponsorUTMs.from_dict(d.pop("utm"))




        button = d.pop("button", UNSET)

        headline = d.pop("headline", UNSET)

        image = d.pop("image", UNSET)

        link = d.pop("link", UNSET)

        logo = d.pop("logo", UNSET)

        text = d.pop("text", UNSET)

        sponsor_creative = cls(
            utm=utm,
            button=button,
            headline=headline,
            image=image,
            link=link,
            logo=logo,
            text=text,
        )

        return sponsor_creative

