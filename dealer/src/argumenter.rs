// Helps building SQL query strings by generating a positional argument for every parameter added
pub struct Argumenter<T> {
    pub values: Vec<T>,
}

impl<T> Argumenter<T> {
    pub fn add(&mut self, v: T) -> String {
        self.values.push(v);
        format!("${}", self.values.len())
    }

    pub fn values(&self) -> &Vec<T> {
        &self.values
    }
}


#[test]
fn number_and_order_of_positional_args_is_correct() {
    let mut arger = Argumenter{values: Vec::<String>::new()};
    let dollar_1 = arger.add("first".to_owned());
    let dollar_2 = arger.add("second".to_owned());
    let dollar_3 = arger.add("thirst".to_owned());

    assert_eq!(dollar_1, "$1");
    assert_eq!(dollar_2, "$2");
    assert_eq!(dollar_3, "$3");
}

#[test]
fn number_and_order_of_args_is_correct() {
    let mut arger = Argumenter{values: Vec::<String>::new()};
    arger.add("first".to_owned());
    arger.add("second".to_owned());
    arger.add("thirst".to_owned());

    let values =  arger.values();
    assert_eq!(values.len(), 3);
    assert_eq!(values[2], "thirst");
    assert_eq!(values[1], "second");
    assert_eq!(values[0], "first");
}
