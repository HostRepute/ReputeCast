from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.integration_site_form_kind import IntegrationSiteFormKind
from typing import cast

if TYPE_CHECKING:
  from ..models.word_press_form_place import WordPressFormPlace





T = TypeVar("T", bound="IntegrationSiteForm")



@_attrs_define
class IntegrationSiteForm:
    """ 
        Attributes:
            allowed (bool): The site is one of the form's allowed domains
            id (str): The form's uuid, for links
            kind (IntegrationSiteFormKind):
            missing (bool): The site uses a form that no longer exists
            name (str):
            places (list[WordPressFormPlace]):
            public_id (str):  Example: frm_8c2e1a.
     """

    allowed: bool
    id: str
    kind: IntegrationSiteFormKind
    missing: bool
    name: str
    places: list[WordPressFormPlace]
    public_id: str





    def to_dict(self) -> dict[str, Any]:
        from ..models.word_press_form_place import WordPressFormPlace # noqa: PLC0415
        allowed = self.allowed

        id = self.id

        kind = self.kind.value

        missing = self.missing

        name = self.name

        places = []
        for places_item_data in self.places:
            places_item = places_item_data.to_dict()
            places.append(places_item)



        public_id = self.public_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "allowed": allowed,
            "id": id,
            "kind": kind,
            "missing": missing,
            "name": name,
            "places": places,
            "publicId": public_id,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.word_press_form_place import WordPressFormPlace # noqa: PLC0415
        d = dict(src_dict)
        allowed = d.pop("allowed")

        id = d.pop("id")

        kind = IntegrationSiteFormKind(d.pop("kind"))




        missing = d.pop("missing")

        name = d.pop("name")

        places = []
        _places = d.pop("places")
        for places_item_data in (_places):
            places_item = WordPressFormPlace.from_dict(places_item_data)



            places.append(places_item)


        public_id = d.pop("publicId")

        integration_site_form = cls(
            allowed=allowed,
            id=id,
            kind=kind,
            missing=missing,
            name=name,
            places=places,
            public_id=public_id,
        )

        return integration_site_form

